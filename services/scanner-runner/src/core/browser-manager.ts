/**
 * Browser Manager
 *
 * Playwright browser management for scanner operations.
 */

import {
	type Browser,
	type BrowserContext,
	type ElementHandle,
	type Page,
	type Route,
	chromium
} from 'playwright';

import { createLogger } from '../utils/logger';
import { resolvePlaywrightImageChromiumExecutablePath } from '../utils/playwright';
import { type TargetValidationPolicy, validateTargetURLForPolicy } from './target-validation';
import {
	type BrowserConfig,
	DEFAULT_WAIT_STRATEGY,
	type PreScanAction,
	type PreScanActionValue,
	type ScannerLogger,
	type WaitStrategy
} from './types';

// Selector-free auth. When the playground operator leaves a login-form field
// override blank, the recipe carries `auto:<field>` instead of a CSS selector.
// The executor resolves these to a real element via login-form heuristics
// (see findLoginFieldInPage) rather than handing them to Playwright's selector
// engine. Every other layer treats the selector as an opaque string.
const AUTO_SELECTOR_PREFIX = 'auto:';
const AUTO_SELECTOR_POLL_INTERVAL_MS = 250;
const SENSITIVE_FIELD_MASK_CSS = `
input[type="password"],
input[autocomplete="current-password"],
input[autocomplete="new-password"],
input[autocomplete="one-time-code"],
[data-stageflow-sensitive="true"] {
  color: transparent !important;
  caret-color: transparent !important;
  text-shadow: 0 0 8px rgba(0, 0, 0, 0.95) !important;
  -webkit-text-security: disc !important;
}
`;

const DEFAULT_BROWSER_CONFIG: BrowserConfig = {
	headless: true,
	args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage', '--disable-gpu'],
	defaultViewport: { width: 1280, height: 720 },
	deviceScaleFactor: 2,
	defaultTimeout: 30_000,
	pageLoadTimeout: 15_000,
	bypassCSP: false
};

function isHTTPURL(rawURL: string): boolean {
	try {
		const url = new URL(rawURL);
		return url.protocol === 'http:' || url.protocol === 'https:';
	} catch {
		return false;
	}
}

/**
 * Manages Playwright browser instances for scanner operations.
 */
export class BrowserManager {
	private readonly navigationStatuses = new WeakMap<Page, number>();

	getNavigationStatus(page: Page): number | undefined {
		return this.navigationStatuses.get(page);
	}
	private browser: Browser | null = null;
	private readonly config: BrowserConfig;
	private readonly pageRedactors = new WeakMap<Page, (value: string) => string>();

	private readonly logger: ScannerLogger;
	private readonly runtimeValidationRoutes = new WeakSet<Page>();
	private readonly blockedNavigationErrors = new WeakMap<Page, Error>();

	constructor(config?: Partial<BrowserConfig>, logger?: ScannerLogger) {
		this.config = { ...DEFAULT_BROWSER_CONFIG, ...config };
		this.logger = logger ?? createLogger('BrowserManager');
	}

	async launch(): Promise<Browser> {
		if (this.browser) {
			return this.browser;
		}

		this.browser = await this.launchChromium();

		return this.browser;
	}

	/**
	 * Launch Chromium. The hardened container needs several fallbacks: a
	 * bundled-Chrome executablePath when the default download is missing, and a
	 * single-process retry for constrained sandboxes.
	 */
	private async launchChromium(): Promise<Browser> {
		const fallbackExecutable = resolvePlaywrightImageChromiumExecutablePath();

		const launchAttempts = [
			{ name: 'default', opts: {} },
			{
				name: 'fallback-chrome',
				opts: fallbackExecutable ? { executablePath: fallbackExecutable } : null
			},
			{
				name: 'single-process',
				opts: { args: [...this.config.args, '--single-process'] }
			}
		].filter((attempt) => attempt.opts !== null) as {
			name: string;
			opts: Record<string, unknown>;
		}[];

		let lastError: unknown;
		for (const attempt of launchAttempts) {
			try {
				const attemptArgs = Array.isArray(attempt.opts.args)
					? (attempt.opts.args as string[])
					: this.config.args;
				const executablePath =
					typeof attempt.opts.executablePath === 'string' ? attempt.opts.executablePath : undefined;

				this.logger.info('Launching browser', {
					attempt: attempt.name,
					headless: this.config.headless,
					args: attempt.opts.args ?? this.config.args
				});

				return await chromium.launch({
					headless: this.config.headless,
					args: attemptArgs,
					chromiumSandbox: false,
					env: {
						...process.env,
						DBUS_SESSION_BUS_ADDRESS: 'disabled'
					},
					...(executablePath !== undefined ? { executablePath } : {})
				});
			} catch (err) {
				lastError = err;
				this.logger.warn('Browser launch attempt failed', {
					attempt: attempt.name,
					error: err instanceof Error ? err.message : String(err)
				});
			}
		}

		throw lastError instanceof Error ? lastError : new Error(String(lastError));
	}

	async createContext(
		viewport?: { width: number; height: number },
		options?: { storageStatePath?: string }
	): Promise<BrowserContext> {
		const browser = await this.launch();

		const vp = viewport ?? this.config.defaultViewport;

		const storageStatePath = options?.storageStatePath;
		if (storageStatePath !== undefined) {
			this.logger.info('Creating browser context with storage state', {
				storageStatePath
			});
		}

		return browser.newContext({
			viewport: vp,
			deviceScaleFactor: this.config.deviceScaleFactor,
			// We inject small amounts of CSS/JS for scanning + screenshots; some sites have strict CSP that would block it.
			bypassCSP: this.config.bypassCSP ?? false,
			...(storageStatePath !== undefined ? { storageState: storageStatePath } : {})
		});
	}

	async navigateToPage(
		page: Page,
		url: string,
		waitStrategy?: WaitStrategy,
		targetValidationPolicy: TargetValidationPolicy = { allowedOrigins: [] },
		redact: (value: string) => string = this.pageRedactors.get(page) ?? ((value) => value)
	): Promise<void> {
		this.pageRedactors.set(page, redact);
		await this.ensureRuntimeTargetValidationRouting(page, targetValidationPolicy, redact);
		this.blockedNavigationErrors.delete(page);
		await validateTargetURLForPolicy(url, targetValidationPolicy);

		const strategy = waitStrategy ?? DEFAULT_WAIT_STRATEGY;

		this.logger.debug('Navigating to page', {
			url: redact(url),
			waitStrategy: strategy.type
		});

		const waitUntil = this.getPlaywrightWaitUntil(strategy);

		try {
			const response = await page.goto(url, {
				waitUntil,
				timeout: this.config.pageLoadTimeout
			});
			if (response) {
				this.navigationStatuses.set(page, response.status());
			}
		} catch (err) {
			const blockedError = this.blockedNavigationErrors.get(page);
			if (blockedError) {
				this.blockedNavigationErrors.delete(page);
				throw blockedError;
			}

			throw err;
		}

		// Clear any stale route-captured errors from subframe navigations.
		this.blockedNavigationErrors.delete(page);

		// Validate the final navigated URL (after redirects) as a belt-and-suspenders check.
		const finalURL = page.url();
		if (finalURL && finalURL !== url) {
			await validateTargetURLForPolicy(finalURL, targetValidationPolicy);
		}

		await this.applyAdditionalWait(page, strategy);
	}

	private async ensureRuntimeTargetValidationRouting(
		page: Page,
		targetValidationPolicy: TargetValidationPolicy,
		redact: (value: string) => string
	): Promise<void> {
		if (this.runtimeValidationRoutes.has(page)) {
			return;
		}

		this.runtimeValidationRoutes.add(page);

		await page.route('**/*', async (route: Route) => {
			const request = route.request();
			const requestURL = request.url();
			const isNavigation = request.resourceType() === 'document' && request.isNavigationRequest();

			if (!isHTTPURL(requestURL)) {
				await route.continue();
				return;
			}

			try {
				await validateTargetURLForPolicy(requestURL, targetValidationPolicy);
				await route.continue();
			} catch (err) {
				const error = err instanceof Error ? err : new Error(String(err));
				if (isNavigation) {
					this.blockedNavigationErrors.set(page, error);
				}

				this.logger.warn('Blocked browser request', {
					resourceType: request.resourceType(),
					url: redact(requestURL),
					error: redact(error.message)
				});

				await route.abort('blockedbyclient');
			}
		});
	}

	async executePreScanActions(
		page: Page,
		actions: PreScanAction[],
		options?: { maskInputValues?: boolean }
	): Promise<void> {
		if (actions.length === 0) {
			return;
		}

		this.logger.debug('Executing pre-scan actions', { count: actions.length });

		for (const action of actions) {
			await this.executeAction(page, action, options?.maskInputValues === true);
		}
	}

	private async executeAction(
		page: Page,
		action: PreScanAction,
		maskInputValue: boolean
	): Promise<void> {
		const timeout =
			'timeout' in action && typeof action.timeout === 'number'
				? action.timeout
				: this.config.defaultTimeout;

		// Resolve the selector-free auth convention before falling through to
		// Playwright's selector engine.
		if (
			'selector' in action &&
			typeof action.selector === 'string' &&
			action.selector.startsWith(AUTO_SELECTOR_PREFIX)
		) {
			await this.executeAutoLoginAction(page, action, timeout, maskInputValue);
			return;
		}

		switch (action.type) {
			case 'click':
				await page.click(action.selector, { timeout });
				return;
			case 'fill': {
				if (maskInputValue) {
					await this.markSensitiveField(page, action.selector);
				}
				const value = resolveActionValue(action.value);
				await page.fill(action.selector, value, { timeout });
				return;
			}
			case 'select': {
				if (maskInputValue) {
					await this.markSensitiveField(page, action.selector);
				}
				const value = resolveActionValue(action.value);
				await page.selectOption(action.selector, value, { timeout });
				return;
			}
			case 'hover':
				await page.hover(action.selector, { timeout });
				return;
			case 'wait':
				await page.waitForTimeout(action.ms);
				return;
			case 'scroll':
				if (action.selector) {
					const element = page.locator(action.selector);
					await element.scrollIntoViewIfNeeded({ timeout });
					return;
				}

				await page.evaluate(
					(scrollY) => {
						window.scrollBy(0, scrollY);
					},
					(action.direction === 'up' ? -1 : 1) * (action.pixels ?? 500)
				);
				return;
			case 'keyboard':
				await page.keyboard.press(action.key);
				return;
			default:
				this.logger.warn('Unknown action type', { action });
				return;
		}
	}

	/**
	 * Handles an `auto:<field>` step: locate the login-form element by heuristic,
	 * then fill or click it. `auto:submit` falls back to pressing Enter when no
	 * submit control is found, which submits the form the password field is in.
	 */
	private async executeAutoLoginAction(
		page: Page,
		action: PreScanAction,
		timeout: number,
		maskInputValue: boolean
	): Promise<void> {
		if (action.type !== 'fill' && action.type !== 'click') {
			throw new Error(
				`The auto-detect selector convention supports only fill and click steps (got "${action.type}").`
			);
		}

		const field = action.selector.slice(AUTO_SELECTOR_PREFIX.length);
		const element = await this.resolveAutoLoginElement(page, field, timeout);

		if (action.type === 'fill') {
			if (!element) {
				throw new Error(
					`Auto-detect could not find the "${field}" field on the login form. ` +
						'Open Advanced Settings and provide a CSS selector for this field.'
				);
			}

			try {
				if (maskInputValue) {
					await this.ensureSensitiveFieldMask(page);
					await element.evaluate((node) => {
						node.setAttribute('data-stageflow-sensitive', 'true');
					});
				}
				const value = resolveActionValue(action.value);
				await element.fill(value, { timeout });
			} finally {
				await element.dispose();
			}
			return;
		}

		// click — typically auto:submit.
		if (!element) {
			this.logger.debug('Auto-detect found no submit control; pressing Enter to submit', { field });
			await page.keyboard.press('Enter');
			return;
		}

		await element.click({ timeout });
		await element.dispose();
	}

	private async markSensitiveField(page: Page, selector: string): Promise<void> {
		await this.ensureSensitiveFieldMask(page);
		await page.locator(selector).evaluate((node) => {
			node.setAttribute('data-stageflow-sensitive', 'true');
		});
	}

	private async ensureSensitiveFieldMask(page: Page): Promise<void> {
		if (typeof page.addStyleTag !== 'function') {
			throw new Error('Sensitive-field masking is unavailable; refusing to continue');
		}

		try {
			await page.addStyleTag({ content: SENSITIVE_FIELD_MASK_CSS });
			return;
		} catch {
			// A `style-src` policy without 'unsafe-inline' blocks the <style> element.
			// Constructed stylesheets are CSSOM, which CSP does not govern.
		}

		try {
			await page.evaluate((css) => {
				const sheet = new CSSStyleSheet();
				sheet.replaceSync(css);
				document.adoptedStyleSheets = [...document.adoptedStyleSheets, sheet];
			}, SENSITIVE_FIELD_MASK_CSS);
		} catch (error) {
			throw new Error('Failed to install sensitive-field masking; refusing to continue', {
				cause: error
			});
		}
	}

	/**
	 * Polls the page (up to `timeout`) for the requested login field, returning a
	 * handle to the first match or null if none appears. The heuristic itself runs
	 * in the page context — see findLoginFieldInPage.
	 */
	private async resolveAutoLoginElement(
		page: Page,
		field: string,
		timeout: number
	): Promise<ElementHandle<HTMLElement> | null> {
		const deadline = Date.now() + timeout;
		for (;;) {
			const handle = await page.evaluateHandle(findLoginFieldInPage, field);
			const element = handle.asElement();
			if (element) {
				return element;
			}

			await handle.dispose();
			if (Date.now() >= deadline) {
				return null;
			}

			await page.waitForTimeout(AUTO_SELECTOR_POLL_INTERVAL_MS);
		}
	}

	private getPlaywrightWaitUntil(
		strategy: WaitStrategy
	): 'load' | 'domcontentloaded' | 'networkidle' {
		switch (strategy.type) {
			case 'load':
			case 'selector':
			case 'timeout':
				return 'load';
			case 'domcontentloaded':
				return 'domcontentloaded';
			case 'networkidle':
				return 'networkidle';
			default:
				this.logger.warn('Unknown wait strategy for navigation. Falling back to load.', {
					strategy
				});
				return 'load';
		}
	}

	private async applyAdditionalWait(page: Page, strategy: WaitStrategy): Promise<void> {
		switch (strategy.type) {
			case 'load':
			case 'domcontentloaded':
			case 'networkidle':
				return;
			case 'selector':
				await page.waitForSelector(strategy.selector, {
					timeout: strategy.timeout ?? this.config.defaultTimeout
				});
				return;
			case 'timeout':
				await page.waitForTimeout(strategy.ms);
				return;
			default:
				this.logger.warn('Unknown wait strategy for post-navigation wait.', {
					strategy
				});
				return;
		}
	}

	async close(): Promise<void> {
		if (!this.browser) {
			return;
		}

		this.logger.info('Closing browser');
		await this.browser.close();
		this.browser = null;
	}

	getBrowser(): Browser | null {
		return this.browser;
	}

	getConfig(): BrowserConfig {
		return { ...this.config };
	}
}

/**
 * Login-form field heuristic, evaluated in the page context. Given a logical
 * field name (`username` | `password` | `submit`), returns the best-matching
 * visible element, or null. Must be fully self-contained: it is serialized and
 * run inside the browser, so it cannot close over module scope.
 */
function findLoginFieldInPage(field: string): HTMLElement | null {
	const isVisible = (el: Element | null): el is HTMLElement => {
		if (!el) {
			return false;
		}
		const style = window.getComputedStyle(el);
		const rect = el.getBoundingClientRect();
		return (
			style.visibility !== 'hidden' && style.display !== 'none' && rect.width > 0 && rect.height > 0
		);
	};

	const passwordField =
		Array.from(document.querySelectorAll<HTMLInputElement>('input[type="password"]')).find(
			isVisible
		) ?? null;

	if (field === 'password') {
		return passwordField;
	}

	if (field === 'username') {
		const inputs = Array.from(document.querySelectorAll<HTMLInputElement>('input')).filter(
			isVisible
		);
		const usernameLike = (el: HTMLInputElement): boolean => {
			const type = (el.getAttribute('type') ?? 'text').toLowerCase();
			return !['password', 'hidden', 'checkbox', 'radio', 'submit', 'button'].includes(type);
		};
		// Prefer username-like fields that precede the password field; otherwise
		// consider every username-like field on the page.
		const candidates =
			passwordField !== null
				? inputs.slice(0, inputs.indexOf(passwordField)).filter(usernameLike)
				: inputs.filter(usernameLike);
		// An explicit email input wins; otherwise take the one closest to the password.
		const email = candidates.find(
			(el) => (el.getAttribute('type') ?? '').toLowerCase() === 'email'
		);
		return email ?? candidates[candidates.length - 1] ?? null;
	}

	if (field === 'submit') {
		const scope: ParentNode = passwordField?.closest('form') ?? document;
		const explicit = scope.querySelector<HTMLElement>(
			'button[type="submit"], input[type="submit"]'
		);
		if (isVisible(explicit)) {
			return explicit;
		}
		const buttons = Array.from(
			scope.querySelectorAll<HTMLElement>('button, input[type="button"], [role="button"]')
		).filter(isVisible);
		const labelOf = (el: HTMLElement): string =>
			el instanceof HTMLInputElement ? el.value : el.textContent;
		const labelled = buttons.find((el) =>
			/log\s*in|sign\s*in|continue|submit|next|go/i.test(labelOf(el).trim())
		);
		return labelled ?? buttons[0] ?? null;
	}

	return null;
}

function resolveActionValue(value: PreScanActionValue): string {
	if (typeof value !== 'string') {
		throw new Error('Pre-scan action values must be literal strings');
	}
	return value;
}
