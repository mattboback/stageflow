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

import fs from 'fs-extra';
import { dirname, join } from 'node:path';

import type { BrowserManager } from './browser-manager';
import type { SecretsResolver } from './secrets-resolver';
import type { TargetValidationPolicy } from './target-validation';
import type {
	ProvenanceAuth,
	ProvenanceAuthForm,
	ProvenanceAuthStorageState,
	ScannerLogger,
	StorageProvider
} from './types';

export type AuthMode = ProvenanceAuth['mode'];

export interface AuthHydrationErrorOptions {
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

export interface HydrateStorageStateOptions {
	auth: ProvenanceAuthStorageState;
	storageProvider: StorageProvider;
	bucket: string;
	destPath: string;
	logger: ScannerLogger;
}

export interface HydrateStorageStateResult {
	storageStatePath: string;
}

export async function hydrateStorageState(
	options: HydrateStorageStateOptions
): Promise<HydrateStorageStateResult> {
	const { auth, storageProvider, bucket, destPath, logger } = options;

	try {
		await fs.ensureDir(dirname(destPath));
		await storageProvider.download(bucket, auth.artifact_key, destPath);
		await fs.chmod(destPath, 0o600);
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

export interface HydrateFormOptions {
	auth: ProvenanceAuthForm;
	context: BrowserContext;
	browserManager: BrowserManager;
	targetValidationPolicy: TargetValidationPolicy;
	secretsResolver: SecretsResolver;
	logger: ScannerLogger;
}

export interface HydrateFormResult {
	postLoginUrl: string;
}

export async function hydrateForm(options: HydrateFormOptions): Promise<HydrateFormResult> {
	const { auth, context, browserManager, targetValidationPolicy, secretsResolver, logger } =
		options;

	const page = await context.newPage();
	let postLoginUrl: string | undefined;

	try {
		logger.info('Hydrating auth via form recipe', {
			loginUrl: auth.login_url,
			steps: auth.steps.length,
			successType: auth.success.type
		});

		await browserManager.navigateToPage(
			page,
			auth.login_url,
			{ type: 'load' },
			targetValidationPolicy
		);

		await browserManager.executePreScanActions(page, auth.steps, secretsResolver);

		await waitForSuccess(page, auth);

		postLoginUrl = page.url();
		if (await isStillOnVisibleLoginForm(page, auth.login_url, postLoginUrl)) {
			throw new AuthHydrationError(
				`Form auth hydration did not leave the login page: ${postLoginUrl}. ` +
					'Use a success selector that only appears after login or verify the submitted credentials.',
				{
					mode: 'form',
					loginUrl: auth.login_url,
					postLoginUrl
				}
			);
		}

		return { postLoginUrl };
	} catch (err) {
		const capturedUrl = postLoginUrl ?? safeUrl(page);
		if (err instanceof AuthHydrationError) {
			throw err;
		}
		throw new AuthHydrationError(
			`Form auth hydration failed at ${capturedUrl ?? auth.login_url}: ${
				err instanceof Error ? err.message : String(err)
			}`,
			{
				mode: 'form',
				loginUrl: auth.login_url,
				...(capturedUrl !== undefined ? { postLoginUrl: capturedUrl } : {}),
				cause: err
			}
		);
	} finally {
		try {
			await page.close();
		} catch (closeErr) {
			logger.warn('Failed to close auth hydration page', {
				error: closeErr instanceof Error ? closeErr.message : String(closeErr)
			});
		}
	}
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
