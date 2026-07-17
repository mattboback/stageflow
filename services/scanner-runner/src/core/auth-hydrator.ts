/**
 * Authentication hydration for the scanner-runner.
 *
 * Two modes, both designed to run once per scanner before page iteration so
 * every downstream scanner inherits the authenticated context for free:
 *
 *   - `storage_state`: fetch a Playwright storage-state JSON file via the
 *     existing StorageProvider and feed its path into context creation.
 *   - `form`: open a single internal page, replay the recipe steps via the
 *     existing PreScanAction executor (with `from_env` resolved through the
 *     SecretsResolver), wait for the success strategy, then discard the page.
 *
 * Failures throw {@link AuthHydrationError}. PageIterator turns that error
 * into an `auth-hydration-failed` issue at `severity: critical` for every
 * page in the run, then returns without iterating.
 */

import type { BrowserContext } from 'playwright';

import { chmod } from 'node:fs/promises';
import { dirname, join } from 'node:path';

import type { BrowserManager } from './browser-manager';
import type { SecretsResolver } from './secrets-resolver';
import { redactURLUserInfo, type TargetValidationPolicy } from './target-validation';
import type {
	ProvenanceAuth,
	ProvenanceAuthForm,
	ProvenanceAuthStorageState,
	ScannerLogger,
	StorageProvider
} from './types';
import { ensureDir } from '../utils/fs';

type AuthMode = ProvenanceAuth['mode'];

interface AuthHydrationErrorOptions {
	mode: AuthMode;
	loginUrl?: string;
	postLoginUrl?: string;
	cause?: unknown;
}

export class AuthHydrationError extends Error {
	readonly mode: AuthMode;
	readonly loginUrl: string | undefined;
	readonly postLoginUrl: string | undefined;

	constructor(message: string, options: AuthHydrationErrorOptions) {
		const cause = options.cause;
		super(message, cause instanceof Error ? { cause } : undefined);
		this.name = 'AuthHydrationError';
		this.mode = options.mode;
		this.loginUrl = options.loginUrl;
		this.postLoginUrl = options.postLoginUrl;
	}
}

interface HydrateStorageStateOptions {
	auth: ProvenanceAuthStorageState;
	storageProvider: StorageProvider;
	bucket: string;
	destPath: string;
	logger: ScannerLogger;
}

interface HydrateStorageStateResult {
	storageStatePath: string;
}

export async function hydrateStorageState(
	options: HydrateStorageStateOptions
): Promise<HydrateStorageStateResult> {
	const { auth, storageProvider, bucket, destPath, logger } = options;

	try {
		await ensureDir(dirname(destPath));
		await storageProvider.download(bucket, auth.artifact_key, destPath);
		await chmod(destPath, 0o600);
		logger.info('Downloaded auth storage state', { artifactKey: auth.artifact_key });
		return { storageStatePath: destPath };
	} catch (err) {
		throw new AuthHydrationError(
			`Failed to download storage state artifact "${auth.artifact_key}": ${
				err instanceof Error ? err.message : String(err)
			}`,
			{ mode: 'storage_state', cause: err }
		);
	}
}

interface HydrateFormOptions {
	auth: ProvenanceAuthForm;
	context: BrowserContext;
	browserManager: BrowserManager;
	targetValidationPolicy: TargetValidationPolicy;
	secretsResolver: SecretsResolver;
	logger: ScannerLogger;
}

interface HydrateFormResult {
	postLoginUrl: string;
}

export async function hydrateForm(options: HydrateFormOptions): Promise<HydrateFormResult> {
	const { auth, context, browserManager, targetValidationPolicy, secretsResolver, logger } =
		options;

	const page = await context.newPage();
	let capturedPostLoginUrl: string | undefined;
	const redact = (value: string): string => secretsResolver.redactKnownValues(value);
	const sanitizeUrl = (value: string): string => redact(redactURLUserInfo(value));

	try {
		registerFormActionValues(auth, secretsResolver);
		logger.info('Hydrating auth via form recipe', {
			loginUrl: sanitizeUrl(auth.login_url),
			steps: auth.steps.length,
			successType: auth.success.type
		});

		await browserManager.navigateToPage(
			page,
			auth.login_url,
			{ type: 'load' },
			targetValidationPolicy
		);

		await browserManager.executePreScanActions(page, auth.steps, secretsResolver, {
			maskInputValues: true
		});

		await waitForSuccess(page, auth);

		// Confirm we actually left the login form. Client-rendered apps often finish
		// their post-login redirect a beat after the success wait resolves (e.g. a
		// `networkidle` wait completes once the auth XHR settles, before the SPA swaps
		// routes), so poll for a grace period instead of failing on a single snapshot.
		// Selector-based success already waited for real post-login content, so this
		// returns on the first check in that case.
		const leftLoginFormUrl = await waitUntilLeftLoginForm(page, auth.login_url);
		if (leftLoginFormUrl === null) {
			capturedPostLoginUrl = page.url();
			const safePostLoginUrl = sanitizeUrl(capturedPostLoginUrl);
			throw new AuthHydrationError(
				`Form auth hydration did not leave the login page: ${safePostLoginUrl}. ` +
					'Verify the submitted credentials, or set a success selector that only appears after login.',
				{
					mode: 'form',
					loginUrl: sanitizeUrl(auth.login_url),
					postLoginUrl: safePostLoginUrl
				}
			);
		}

		capturedPostLoginUrl = leftLoginFormUrl;
		return { postLoginUrl: sanitizeUrl(capturedPostLoginUrl) };
	} catch (err) {
		const capturedUrl = sanitizeUrl(capturedPostLoginUrl ?? safeUrl(page) ?? auth.login_url);
		if (err instanceof AuthHydrationError) {
			throw sanitizeAuthHydrationError(err, redact);
		}
		const safeErrorMessage = redact(err instanceof Error ? err.message : String(err));
		throw new AuthHydrationError(
			`Form auth hydration failed at ${capturedUrl}: ${safeErrorMessage}`,
			{
				mode: 'form',
				loginUrl: sanitizeUrl(auth.login_url),
				postLoginUrl: capturedUrl,
				cause: new Error(safeErrorMessage)
			}
		);
	} finally {
		try {
			await page.close();
		} catch (closeErr) {
			logger.warn('Failed to close auth hydration page', {
				error: redact(closeErr instanceof Error ? closeErr.message : String(closeErr))
			});
		}
	}
}

function registerFormActionValues(
	auth: ProvenanceAuthForm,
	secretsResolver: SecretsResolver
): void {
	for (const action of auth.steps) {
		if (action.type === 'fill' || action.type === 'select') {
			secretsResolver.resolveValue(action.value);
		}
	}
}

function sanitizeAuthHydrationError(
	error: AuthHydrationError,
	redact: (value: string) => string
): AuthHydrationError {
	const cause = error.cause;
	return new AuthHydrationError(redact(error.message), {
		mode: error.mode,
		...(error.loginUrl !== undefined ? { loginUrl: redact(error.loginUrl) } : {}),
		...(error.postLoginUrl !== undefined ? { postLoginUrl: redact(error.postLoginUrl) } : {}),
		...(cause instanceof Error ? { cause: new Error(redact(cause.message)) } : {})
	});
}

async function waitForSuccess(
	page: import('playwright').Page,
	auth: ProvenanceAuthForm
): Promise<void> {
	const strategy = auth.success;
	switch (strategy.type) {
		case 'load':
			await page.waitForLoadState('load');
			return;
		case 'domcontentloaded':
			await page.waitForLoadState('domcontentloaded');
			return;
		case 'networkidle':
			await page.waitForLoadState('networkidle');
			return;
		case 'selector':
			await page.waitForSelector(strategy.selector, {
				timeout: strategy.timeout ?? 30_000
			});
			return;
		case 'timeout':
			await page.waitForTimeout(strategy.ms);
			return;
		default: {
			const exhaustive: never = strategy;
			throw new Error(
				`Unknown success wait strategy: ${JSON.stringify(exhaustive satisfies never)}`
			);
		}
	}
}

function safeUrl(page: import('playwright').Page): string | undefined {
	try {
		const url = page.url();
		return url ? url : undefined;
	} catch {
		return undefined;
	}
}

/**
 * Default destination for a downloaded storage-state artifact within the
 * scanner's results directory. The file is written with mode 0600 and is
 * cleaned up by ScannerBase.cleanup() at end of run.
 */
export function defaultStorageStatePath(resultsDir: string): string {
	return join(resultsDir, 'auth', 'storage-state.json');
}

// Grace period for a client-side post-login redirect to complete. An SPA route
// swap almost always lands within a tick of the auth XHR resolving (and the
// `networkidle` success wait already absorbed the network quiet period), so this
// is a safety net for slow redirects — not the expected wait. Kept short enough
// that a genuinely failed login surfaces its error promptly rather than hanging.
const LOGIN_FORM_CLEAR_TIMEOUT_MS = 12_000;
const LOGIN_FORM_POLL_INTERVAL_MS = 750;

/**
 * Polls until the page has left the visible login form, returning the post-login
 * URL once it has. Returns null if the login form is still visible after the
 * grace period (i.e. login appears to have failed).
 */
async function waitUntilLeftLoginForm(
	page: import('playwright').Page,
	loginUrl: string
): Promise<string | null> {
	const deadline = Date.now() + LOGIN_FORM_CLEAR_TIMEOUT_MS;
	for (;;) {
		const currentUrl = page.url();
		if (!(await isStillOnVisibleLoginForm(page, loginUrl, currentUrl))) {
			return currentUrl;
		}
		if (Date.now() >= deadline) {
			return null;
		}
		await page.waitForTimeout(LOGIN_FORM_POLL_INTERVAL_MS);
	}
}

async function isStillOnVisibleLoginForm(
	page: import('playwright').Page,
	loginUrl: string,
	postLoginUrl: string
): Promise<boolean> {
	if (!sameOriginAndPath(loginUrl, postLoginUrl)) {
		return false;
	}

	try {
		return await page.evaluate(() => {
			const passwordInput = document.querySelector<HTMLInputElement>('input[type="password"]');
			if (!passwordInput) {
				return false;
			}

			const style = window.getComputedStyle(passwordInput);
			const rect = passwordInput.getBoundingClientRect();
			return (
				style.visibility !== 'hidden' &&
				style.display !== 'none' &&
				rect.width > 0 &&
				rect.height > 0
			);
		});
	} catch {
		return false;
	}
}

function sameOriginAndPath(a: string, b: string): boolean {
	try {
		const left = new URL(a);
		const right = new URL(b);
		return (
			left.origin === right.origin &&
			trimTrailingSlash(left.pathname) === trimTrailingSlash(right.pathname)
		);
	} catch {
		return a === b;
	}
}

function trimTrailingSlash(pathname: string): string {
	if (pathname === '/') {
		return pathname;
	}
	return pathname.replace(/\/+$/, '');
}
