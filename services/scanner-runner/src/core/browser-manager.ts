/**
 * Browser Manager
 *
 * Playwright browser management for scanner operations.
 */

import { type Browser, type BrowserContext, type Page, type Route, chromium } from 'playwright';

import { createLogger } from '../utils/logger';
import { resolvePlaywrightImageChromiumExecutablePath } from '../utils/playwright';
import {
	shouldEnforceRuntimeTargetValidation,
	validateRuntimeTargetURL
} from './target-validation';
import {
	type BrowserConfig,
	DEFAULT_WAIT_STRATEGY,
	type PreScanAction,
	type ScannerLogger,
	type WaitStrategy
} from './types';

const DEFAULT_BROWSER_CONFIG: BrowserConfig = {
	headless: true,
	args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage', '--disable-gpu'],
	defaultViewport: { width: 1280, height: 720 },
	deviceScaleFactor: 2,
	defaultTimeout: 30_000,
	pageLoadTimeout: 15_000,
	bypassCSP: false
};

/**
 * Manages Playwright browser instances for scanner operations.
 */
export class BrowserManager {
	private browser: Browser | null = null;
	private readonly config: BrowserConfig;
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

				this.browser = await chromium.launch({
					headless: this.config.headless,
					args: attemptArgs,
					chromiumSandbox: false,
					env: {
						...process.env,
						DBUS_SESSION_BUS_ADDRESS: 'disabled'
					},
					...(executablePath !== undefined ? { executablePath } : {})
				});

				return this.browser;
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

	async createContext(viewport?: { width: number; height: number }): Promise<BrowserContext> {
		const browser = await this.launch();

		const vp = viewport ?? this.config.defaultViewport;

		return browser.newContext({
			viewport: vp,
			deviceScaleFactor: this.config.deviceScaleFactor,
			// We inject small amounts of CSS/JS for scanning + screenshots; some sites have strict CSP that would block it.
			bypassCSP: this.config.bypassCSP ?? false
		});
	}

	async navigateToPage(page: Page, url: string, waitStrategy?: WaitStrategy): Promise<void> {
		const enforceRuntimeTargetValidation = shouldEnforceRuntimeTargetValidation();
		if (enforceRuntimeTargetValidation) {
			await this.ensureRuntimeTargetValidationRouting(page);
			this.blockedNavigationErrors.delete(page);
			await validateRuntimeTargetURL(url);
		}

		const strategy = waitStrategy ?? DEFAULT_WAIT_STRATEGY;

		this.logger.debug('Navigating to page', {
			url,
			waitStrategy: strategy.type
		});

		const waitUntil = this.getPlaywrightWaitUntil(strategy);

		try {
			await page.goto(url, {
				waitUntil,
				timeout: this.config.pageLoadTimeout
			});
		} catch (err) {
			const blockedError = this.blockedNavigationErrors.get(page);
			if (blockedError) {
				this.blockedNavigationErrors.delete(page);
				throw blockedError;
			}

			throw err;
		}

		if (enforceRuntimeTargetValidation) {
			// Clear any stale route-captured errors from subframe navigations.
			this.blockedNavigationErrors.delete(page);

			// Validate the final navigated URL (after redirects) as a belt-and-suspenders check.
			const finalURL = page.url();
			if (finalURL && finalURL !== url) {
				await validateRuntimeTargetURL(finalURL);
			}
		}

		await this.applyAdditionalWait(page, strategy);
	}

	private async ensureRuntimeTargetValidationRouting(page: Page): Promise<void> {
		if (this.runtimeValidationRoutes.has(page)) {
			return;
		}

		this.runtimeValidationRoutes.add(page);

		await page.route('**/*', async (route: Route) => {
			const request = route.request();

			// Only validate navigation/document requests (includes redirects).
			if (request.resourceType() !== 'document' || !request.isNavigationRequest()) {
				await route.continue();
				return;
			}

			const requestURL = request.url();

			try {
				await validateRuntimeTargetURL(requestURL);
				await route.continue();
			} catch (err) {
				const error = err instanceof Error ? err : new Error(String(err));
				this.blockedNavigationErrors.set(page, error);

				this.logger.warn('Blocked navigation request', {
					url: requestURL,
					error: error.message
				});

				await route.abort('blockedbyclient');
			}
		});
	}

	async executePreScanActions(page: Page, actions: PreScanAction[]): Promise<void> {
		if (actions.length === 0) {
			return;
		}

		this.logger.debug('Executing pre-scan actions', { count: actions.length });

		for (const action of actions) {
			await this.executeAction(page, action);
		}
	}

	private async executeAction(page: Page, action: PreScanAction): Promise<void> {
		const timeout =
			'timeout' in action && typeof action.timeout === 'number'
				? action.timeout
				: this.config.defaultTimeout;

		switch (action.type) {
			case 'click':
				await page.click(action.selector, { timeout });
				return;
			case 'fill':
				await page.fill(action.selector, action.value, { timeout });
				return;
			case 'select':
				await page.selectOption(action.selector, action.value, { timeout });
				return;
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
