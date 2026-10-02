/**
 * Browser Manager Tests
 *
 * Comprehensive tests for Playwright browser management.
 */

import { type Browser, type BrowserContext, type Page, chromium } from 'playwright';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { PreScanAction, WaitStrategy } from '../../src/core/types';

import { BrowserManager } from '../../src/core/browser-manager';
import { validateTargetURLForPolicy } from '../../src/core/target-validation';

// Mock Playwright
vi.mock('playwright', () => ({
	chromium: {
		launch: vi.fn()
	}
}));

vi.mock('../../src/core/target-validation', () => ({
	validateTargetURLForPolicy: vi.fn().mockResolvedValue(undefined)
}));

describe('BrowserManager', () => {
	let browserManager: BrowserManager;
	let mockBrowser: Browser;
	let mockContext: BrowserContext;
	let mockPage: Page;

	beforeEach(() => {
		// The fallback resolver honours PLAYWRIGHT_BROWSERS_PATH before the image
		// default /ms-playwright, which the fallback tests assert. Hide any host value.
		vi.stubEnv('PLAYWRIGHT_BROWSERS_PATH', undefined);

		// Setup mock browser
		mockPage = {
			goto: vi.fn().mockResolvedValue(undefined),
			route: vi.fn().mockResolvedValue(undefined),
			isClosed: vi.fn().mockReturnValue(false),
			url: vi.fn().mockReturnValue(''),
			click: vi.fn().mockResolvedValue(undefined),
			fill: vi.fn().mockResolvedValue(undefined),
			selectOption: vi.fn().mockResolvedValue(undefined),
			hover: vi.fn().mockResolvedValue(undefined),
			waitForTimeout: vi.fn().mockResolvedValue(undefined),
			waitForSelector: vi.fn().mockResolvedValue(undefined),
			addStyleTag: vi.fn().mockResolvedValue(undefined),
			locator: vi.fn().mockReturnValue({
				scrollIntoViewIfNeeded: vi.fn().mockResolvedValue(undefined),
				evaluate: vi.fn().mockResolvedValue(undefined)
			}),
			evaluate: vi.fn().mockResolvedValue(undefined),
			evaluateHandle: vi.fn(),
			keyboard: {
				press: vi.fn().mockResolvedValue(undefined)
			}
		} as unknown as Page;

		mockContext = {
			route: vi.fn().mockResolvedValue(undefined),
			newPage: vi.fn().mockResolvedValue(mockPage),
			close: vi.fn().mockResolvedValue(undefined)
		} as unknown as BrowserContext;

		mockBrowser = {
			newContext: vi.fn().mockResolvedValue(mockContext),
			close: vi.fn().mockResolvedValue(undefined)
		} as unknown as Browser;

		vi.mocked(chromium.launch).mockResolvedValue(mockBrowser);

		browserManager = new BrowserManager();
	});

	afterEach(() => {
		vi.unstubAllEnvs();
		vi.clearAllMocks();
	});

	describe('constructor', () => {
		it('should initialize with default config', () => {
			const manager = new BrowserManager();
			const config = manager.getConfig();

			expect(config.headless).toBe(true);
			expect(config.defaultViewport).toEqual({ width: 1280, height: 720 });
			expect(config.deviceScaleFactor).toBe(2);
			expect(config.defaultTimeout).toBe(30_000);
			expect(config.pageLoadTimeout).toBe(15_000);
		});

		it('should merge custom config with defaults', () => {
			const manager = new BrowserManager({
				headless: false,
				defaultTimeout: 60_000
			});
			const config = manager.getConfig();

			expect(config.headless).toBe(false);
			expect(config.defaultTimeout).toBe(60_000);
			expect(config.deviceScaleFactor).toBe(2); // Should keep default
		});
	});

	describe('launch', () => {
		it('should launch browser successfully', async () => {
			const browser = await browserManager.launch();

			expect(browser).toBe(mockBrowser);
			expect(chromium.launch).toHaveBeenCalledWith(
				expect.objectContaining({
					headless: true,
					chromiumSandbox: false
				})
			);
		});

		it('does not hand the scanner environment to Chromium', async () => {
			process.env.PROVENANCE_AUTH_JSON = '{"secret":"literal"}';
			try {
				await browserManager.launch();
			} finally {
				delete process.env.PROVENANCE_AUTH_JSON;
			}

			const options = vi.mocked(chromium.launch).mock.calls[0]?.[0];
			expect(options?.env).toBeDefined();
			expect(options?.env?.PROVENANCE_AUTH_JSON).toBeUndefined();
			expect(options?.env?.DBUS_SESSION_BUS_ADDRESS).toBe('disabled');
		});

		it('should return existing browser if already launched', async () => {
			const browser1 = await browserManager.launch();
			const browser2 = await browserManager.launch();

			expect(browser1).toBe(browser2);
			expect(chromium.launch).toHaveBeenCalledTimes(1);
		});

		it('should throw error if the launch fails', async () => {
			vi.mocked(chromium.launch).mockRejectedValue(new Error('Launch failed'));

			await expect(browserManager.launch()).rejects.toThrow('Launch failed');
			expect(chromium.launch).toHaveBeenCalledTimes(1);
		});
	});

	describe('createContext', () => {
		it('should create browser context with default viewport', async () => {
			const context = await browserManager.createContext();

			expect(context).toBe(mockContext);
			expect(mockBrowser.newContext).toHaveBeenCalledWith({
				viewport: { width: 1280, height: 720 },
				deviceScaleFactor: 2,
				bypassCSP: false
			});
		});

		it('should create browser context with custom viewport', async () => {
			const customViewport = { width: 1920, height: 1080 };
			const context = await browserManager.createContext(customViewport);

			expect(context).toBe(mockContext);
			expect(mockBrowser.newContext).toHaveBeenCalledWith({
				viewport: customViewport,
				deviceScaleFactor: 2,
				bypassCSP: false
			});
		});

		it('should allow CSP bypass when explicitly configured', async () => {
			const manager = new BrowserManager({ bypassCSP: true });
			const context = await manager.createContext();

			expect(context).toBe(mockContext);
			expect(mockBrowser.newContext).toHaveBeenCalledWith({
				viewport: { width: 1280, height: 720 },
				deviceScaleFactor: 2,
				bypassCSP: true
			});
		});

		it('should launch browser if not already launched', async () => {
			await browserManager.createContext();

			expect(chromium.launch).toHaveBeenCalledTimes(1);
		});
	});

	describe('navigateToPage', () => {
		it('should navigate with default wait strategy', async () => {
			await browserManager.navigateToPage(mockPage, 'https://example.com');

			expect(validateTargetURLForPolicy).toHaveBeenCalledWith('https://example.com', {
				allowedOrigins: []
			});
			expect(mockPage.route).toHaveBeenCalled();
			expect(mockPage.goto).toHaveBeenCalledWith(
				'https://example.com',
				expect.objectContaining({
					waitUntil: 'load',
					timeout: 15_000
				})
			);
		});

		it('should navigate with domcontentloaded strategy', async () => {
			const strategy: WaitStrategy = { type: 'domcontentloaded' };
			await browserManager.navigateToPage(mockPage, 'https://example.com', strategy);

			expect(mockPage.goto).toHaveBeenCalledWith(
				'https://example.com',
				expect.objectContaining({
					waitUntil: 'domcontentloaded'
				})
			);
		});

		it('should navigate with networkidle strategy', async () => {
			const strategy: WaitStrategy = { type: 'networkidle' };
			await browserManager.navigateToPage(mockPage, 'https://example.com', strategy);

			expect(mockPage.goto).toHaveBeenCalledWith(
				'https://example.com',
				expect.objectContaining({
					waitUntil: 'networkidle'
				})
			);
		});

		it('should wait for selector after navigation', async () => {
			const strategy: WaitStrategy = { type: 'selector', selector: '#main' };
			await browserManager.navigateToPage(mockPage, 'https://example.com', strategy);

			expect(mockPage.goto).toHaveBeenCalled();
			expect(mockPage.waitForSelector).toHaveBeenCalledWith(
				'#main',
				expect.objectContaining({
					timeout: 30_000
				})
			);
		});

		it('should apply timeout wait strategy', async () => {
			const strategy: WaitStrategy = { type: 'timeout', ms: 5000 };
			await browserManager.navigateToPage(mockPage, 'https://example.com', strategy);

			expect(mockPage.goto).toHaveBeenCalled();
			expect(mockPage.waitForTimeout).toHaveBeenCalledWith(5000);
		});

		it('does not require style injection for unauthenticated navigation', async () => {
			vi.mocked(mockPage.addStyleTag).mockRejectedValueOnce(
				new Error('style-src blocks inline styles')
			);

			await expect(
				browserManager.navigateToPage(mockPage, 'https://example.com')
			).resolves.toBeUndefined();
			expect(mockPage.addStyleTag).not.toHaveBeenCalled();
		});

		it('should fail before navigation when target validation fails', async () => {
			vi.mocked(validateTargetURLForPolicy).mockRejectedValueOnce(new Error('Blocked target URL'));

			await expect(browserManager.navigateToPage(mockPage, 'https://127.0.0.1')).rejects.toThrow(
				'Blocked target URL'
			);
			expect(mockPage.goto).not.toHaveBeenCalled();
		});

		it('should validate navigations via Playwright route handler', async () => {
			let routeHandler:
				| ((route: {
						request: () => {
							resourceType: () => string;
							isNavigationRequest: () => boolean;
							url: () => string;
						};
						continue: () => Promise<void>;
						abort: () => Promise<void>;
				  }) => Promise<void>)
				| undefined;

			const mockRoute = vi.fn().mockImplementation((_pattern, handler) => {
				routeHandler = handler;
			});

			// Make the initial URL pass validation.
			vi.mocked(validateTargetURLForPolicy).mockResolvedValue(undefined);

			const page = {
				...mockPage,
				route: mockRoute,
				url: vi.fn().mockReturnValue('https://example.com')
			} as unknown as Page;

			await browserManager.navigateToPage(page, 'https://example.com');

			expect(routeHandler).toBeDefined();

			await routeHandler?.({
				request: () => ({
					resourceType: () => 'document',
					isNavigationRequest: () => true,
					url: () => 'https://example.com/redirected'
				}),
				continue: vi.fn().mockResolvedValue(undefined),
				abort: vi.fn().mockResolvedValue(undefined)
			});

			expect(validateTargetURLForPolicy).toHaveBeenCalledWith('https://example.com/redirected', {
				allowedOrigins: []
			});
		});

		it('should validate allowed subresource requests via Playwright route handler', async () => {
			let routeHandler:
				| ((route: {
						request: () => {
							resourceType: () => string;
							isNavigationRequest: () => boolean;
							url: () => string;
						};
						continue: () => Promise<void>;
						abort: (reason?: string) => Promise<void>;
				  }) => Promise<void>)
				| undefined;

			const mockRoute = vi.fn().mockImplementation((_pattern, handler) => {
				routeHandler = handler;
			});

			vi.mocked(validateTargetURLForPolicy).mockResolvedValue(undefined);

			const page = {
				...mockPage,
				route: mockRoute,
				url: vi.fn().mockReturnValue('https://example.com')
			} as unknown as Page;

			await browserManager.navigateToPage(page, 'https://example.com');

			expect(routeHandler).toBeDefined();

			const continueRequest = vi.fn().mockResolvedValue(undefined);
			const abort = vi.fn().mockResolvedValue(undefined);

			await routeHandler?.({
				request: () => ({
					resourceType: () => 'image',
					isNavigationRequest: () => false,
					url: () => 'https://cdn.example.com/logo.png'
				}),
				continue: continueRequest,
				abort
			});

			expect(validateTargetURLForPolicy).toHaveBeenCalledWith('https://cdn.example.com/logo.png', {
				allowedOrigins: []
			});
			expect(continueRequest).toHaveBeenCalled();
			expect(abort).not.toHaveBeenCalled();
		});

		it('should abort blocked subresource requests without failing the page navigation', async () => {
			let routeHandler:
				| ((route: {
						request: () => {
							resourceType: () => string;
							isNavigationRequest: () => boolean;
							url: () => string;
						};
						continue: () => Promise<void>;
						abort: (reason?: string) => Promise<void>;
				  }) => Promise<void>)
				| undefined;

			const mockRoute = vi.fn().mockImplementation((_pattern, handler) => {
				routeHandler = handler;
			});

			vi.mocked(validateTargetURLForPolicy).mockResolvedValue(undefined);

			const page = {
				...mockPage,
				route: mockRoute,
				url: vi.fn().mockReturnValue('https://example.com')
			} as unknown as Page;

			await expect(
				browserManager.navigateToPage(page, 'https://example.com')
			).resolves.toBeUndefined();

			expect(routeHandler).toBeDefined();

			vi.mocked(validateTargetURLForPolicy).mockRejectedValueOnce(new Error('Blocked subresource'));

			const continueRequest = vi.fn().mockResolvedValue(undefined);
			const abort = vi.fn().mockResolvedValue(undefined);

			await routeHandler?.({
				request: () => ({
					resourceType: () => 'image',
					isNavigationRequest: () => false,
					url: () => 'http://169.254.169.254/latest/meta-data'
				}),
				continue: continueRequest,
				abort
			});

			expect(continueRequest).not.toHaveBeenCalled();
			expect(abort).toHaveBeenCalledWith('blockedbyclient');
		});

		it('does not abort again or label a continuation error as a policy block', async () => {
			let handler: (route: unknown) => Promise<void> = () =>
				Promise.reject(new Error('route handler not registered'));
			vi.mocked(mockPage.route).mockImplementation((_pattern, callback) => {
				handler = callback as typeof handler;
				return Promise.resolve();
			});
			const logger = { info: vi.fn(), debug: vi.fn(), warn: vi.fn(), error: vi.fn() };
			const manager = new BrowserManager(undefined, logger);
			await manager.navigateToPage(mockPage, 'https://example.com');
			const error = new Error('Unexpected protocol failure');
			const abort = vi.fn().mockResolvedValue(undefined);
			await handler({
				request: () => ({
					url: () => 'https://example.com/image.png',
					resourceType: () => 'image',
					isNavigationRequest: () => false,
					failure: () => null
				}),
				continue: vi.fn().mockRejectedValue(error),
				abort
			});
			expect(abort).not.toHaveBeenCalled();
			expect(logger.warn).not.toHaveBeenCalledWith('Blocked browser request', expect.anything());
			expect(() => {
				manager.assertPageHealthy(mockPage);
			}).toThrow('Unexpected protocol failure');
		});

		it('does not treat a cancelled request as a failed audit', async () => {
			let handler: (route: unknown) => Promise<void> = () =>
				Promise.reject(new Error('route handler not registered'));
			vi.mocked(mockPage.route).mockImplementation((_pattern, callback) => {
				handler = callback as typeof handler;
				return Promise.resolve();
			});
			await browserManager.navigateToPage(mockPage, 'https://example.com');
			const abort = vi.fn();
			await handler({
				request: () => ({
					url: () => 'https://example.com/image.png',
					resourceType: () => 'image',
					isNavigationRequest: () => false,
					failure: () => ({ errorText: 'net::ERR_ABORTED' })
				}),
				continue: vi.fn().mockRejectedValue(new Error('Invalid InterceptionId.')),
				abort
			});
			expect(abort).not.toHaveBeenCalled();
			expect(() => {
				browserManager.assertPageHealthy(mockPage);
			}).not.toThrow();
		});

		it('should abort and surface blocked navigation from route handler', async () => {
			let routeHandler:
				| ((route: {
						request: () => {
							resourceType: () => string;
							isNavigationRequest: () => boolean;
							url: () => string;
						};
						continue: () => Promise<void>;
						abort: (reason?: string) => Promise<void>;
				  }) => Promise<void>)
				| undefined;

			const mockRoute = vi.fn().mockImplementation((_pattern, handler) => {
				routeHandler = handler;
			});

			// First call is for the initial explicit validateTargetURLForPolicy(url) in navigateToPage.
			// We'll trigger the blocked redirect via the route handler after that completes.
			vi.mocked(validateTargetURLForPolicy).mockResolvedValueOnce(undefined);

			// Keep navigation pending until we simulate the redirect via the route handler.
			let gotoReject: ((err: Error) => void) | undefined;
			const gotoPromise = new Promise<void>((_resolve, reject) => {
				gotoReject = reject;
			});

			const page = {
				...mockPage,
				route: mockRoute,
				url: vi.fn().mockReturnValue('https://example.com'),
				goto: vi.fn().mockReturnValue(gotoPromise)
			} as unknown as Page;

			const navPromise = browserManager.navigateToPage(page, 'https://example.com');

			expect(routeHandler).toBeDefined();

			// Let navigateToPage finish its initial runtime validation and start awaiting goto.
			await Promise.resolve();
			await Promise.resolve();

			// Route handler sees a blocked redirect target.
			vi.mocked(validateTargetURLForPolicy).mockRejectedValueOnce(new Error('Blocked redirect'));

			const abort = vi.fn().mockResolvedValue(undefined);
			await routeHandler?.({
				request: () => ({
					resourceType: () => 'document',
					isNavigationRequest: () => true,
					url: () => 'https://127.0.0.1/'
				}),
				continue: vi.fn().mockResolvedValue(undefined),
				abort
			});

			gotoReject?.(new Error('net::ERR_BLOCKED_BY_CLIENT'));

			await expect(navPromise).rejects.toThrow('Blocked redirect');
			expect(abort).toHaveBeenCalled();
		});
	});

	it('redacts credential-bearing navigation URLs in browser logs', async () => {
		const logger = { debug: vi.fn(), info: vi.fn(), warn: vi.fn(), error: vi.fn() };
		const manager = new BrowserManager(undefined, logger);
		const secret = 'browser-navigation-canary';
		await manager.navigateToPage(
			mockPage,
			`https://example.com/?value=${secret}`,
			undefined,
			{ allowedOrigins: [] },
			(value) => value.replaceAll(secret, '[REDACTED]')
		);
		await manager.navigateToPage(mockPage, `https://example.com/again?value=${secret}`);

		expect(JSON.stringify(logger.debug.mock.calls)).not.toContain(secret);
		expect(logger.debug).toHaveBeenCalledWith(
			'Navigating to page',
			expect.objectContaining({ url: 'https://example.com/?value=[REDACTED]' })
		);
	});

	describe('executePreScanActions', () => {
		it('should execute click action', async () => {
			const actions: PreScanAction[] = [{ type: 'click', selector: '#button' }];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.click).toHaveBeenCalledWith('#button', {
				timeout: 30_000
			});
		});

		it('should execute fill action', async () => {
			const actions: PreScanAction[] = [{ type: 'fill', selector: '#input', value: 'test value' }];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.fill).toHaveBeenCalledWith('#input', 'test value', {
				timeout: 30_000
			});
		});

		it('rejects object action values without consulting scanner environment', async () => {
			await expect(
				browserManager.executePreScanActions(mockPage, [
					{ type: 'fill', selector: '#password', value: { from_env: 'PATH' } }
				] as never)
			).rejects.toThrow('Pre-scan action values must be literal strings');
			expect(mockPage.fill).not.toHaveBeenCalled();
		});

		it('marks execution-only input values so screenshots mask them', async () => {
			const evaluate = vi.fn().mockResolvedValue(undefined);
			vi.mocked(mockPage.locator).mockReturnValue({ evaluate } as never);

			await browserManager.executePreScanActions(
				mockPage,
				[{ type: 'fill', selector: '#account-password', value: 'literal-canary' }],
				{ maskInputValues: true }
			);

			expect(mockPage.addStyleTag).toHaveBeenCalledWith({
				content: expect.stringContaining('[data-stageflow-sensitive="true"]')
			});
			expect(evaluate).toHaveBeenCalled();
		});

		it('fails closed when a sensitive input cannot be marked', async () => {
			const inputCanary = 'mask-failure-input-canary-8e31';
			const evaluate = vi.fn().mockRejectedValue(new Error('detached input'));
			vi.mocked(mockPage.locator).mockReturnValue({ evaluate } as never);

			await expect(
				browserManager.executePreScanActions(
					mockPage,
					[{ type: 'fill', selector: '#account-password', value: inputCanary }],
					{ maskInputValues: true }
				)
			).rejects.toThrow('detached input');
			expect(mockPage.fill).not.toHaveBeenCalled();
		});

		it('masks through a constructed stylesheet when CSP blocks the style element', async () => {
			vi.mocked(mockPage.addStyleTag).mockRejectedValueOnce(
				new Error('style-src blocks inline styles')
			);

			await browserManager.executePreScanActions(
				mockPage,
				[{ type: 'fill', selector: '#account-password', value: 'csp-fallback-canary-9d2e' }],
				{ maskInputValues: true }
			);

			expect(mockPage.evaluate).toHaveBeenCalledWith(
				expect.any(Function),
				expect.stringContaining('input[type="password"]')
			);
			expect(mockPage.fill).toHaveBeenCalled();
		});

		it('does not write a sensitive value when mask CSS injection fails', async () => {
			const inputCanary = 'mask-css-input-canary-3b17';
			vi.mocked(mockPage.addStyleTag).mockRejectedValueOnce(
				new Error('style-src blocks inline styles')
			);
			vi.mocked(mockPage.evaluate).mockRejectedValueOnce(new Error('execution context destroyed'));

			await expect(
				browserManager.executePreScanActions(
					mockPage,
					[{ type: 'fill', selector: '#account-password', value: inputCanary }],
					{ maskInputValues: true }
				)
			).rejects.toThrow('Failed to install sensitive-field masking; refusing to continue');
			expect(mockPage.locator).not.toHaveBeenCalled();
			expect(mockPage.fill).not.toHaveBeenCalled();
		});

		it('should execute select action', async () => {
			const actions: PreScanAction[] = [
				{ type: 'select', selector: '#dropdown', value: 'option1' }
			];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.selectOption).toHaveBeenCalledWith('#dropdown', 'option1', {
				timeout: 30_000
			});
		});

		it('should execute hover action', async () => {
			const actions: PreScanAction[] = [{ type: 'hover', selector: '#menu' }];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.hover).toHaveBeenCalledWith('#menu', { timeout: 30_000 });
		});

		it('should execute wait action', async () => {
			const actions: PreScanAction[] = [{ type: 'wait', ms: 2000 }];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.waitForTimeout).toHaveBeenCalledWith(2000);
		});

		it('should execute scroll action with selector', async () => {
			const actions: PreScanAction[] = [{ type: 'scroll', selector: '#footer' }];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.locator).toHaveBeenCalledWith('#footer');
		});

		it('should execute scroll action without selector', async () => {
			const actions: PreScanAction[] = [{ type: 'scroll', direction: 'down', pixels: 300 }];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.evaluate).toHaveBeenCalled();
		});

		it('should execute scroll up action', async () => {
			const actions: PreScanAction[] = [{ type: 'scroll', direction: 'up', pixels: 200 }];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.evaluate).toHaveBeenCalled();
		});

		it('should execute keyboard action', async () => {
			const actions: PreScanAction[] = [{ type: 'keyboard', key: 'Enter' }];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.keyboard.press).toHaveBeenCalledWith('Enter');
		});

		it('should execute multiple actions in sequence', async () => {
			const actions: PreScanAction[] = [
				{ type: 'click', selector: '#button1' },
				{ type: 'fill', selector: '#input', value: 'test' },
				{ type: 'click', selector: '#button2' }
			];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.click).toHaveBeenCalledTimes(2);
			expect(mockPage.fill).toHaveBeenCalledTimes(1);
		});

		it('should respect custom timeout in action', async () => {
			const actions: PreScanAction[] = [{ type: 'click', selector: '#button', timeout: 60_000 }];

			await browserManager.executePreScanActions(mockPage, actions);

			expect(mockPage.click).toHaveBeenCalledWith('#button', {
				timeout: 60_000
			});
		});

		it('should do nothing with empty actions array', async () => {
			await browserManager.executePreScanActions(mockPage, []);

			expect(mockPage.click).not.toHaveBeenCalled();
		});

		describe('auto-detect login selectors', () => {
			// A fake ElementHandle plus the evaluateHandle wrapper the resolver unwraps.
			function fakeHandle(element: unknown) {
				return {
					asElement: () => element,
					dispose: vi.fn().mockResolvedValue(undefined)
				};
			}
			function fakeElement() {
				return {
					fill: vi.fn().mockResolvedValue(undefined),
					click: vi.fn().mockResolvedValue(undefined),
					evaluate: vi.fn().mockResolvedValue(undefined),
					dispose: vi.fn().mockResolvedValue(undefined)
				};
			}

			it('fills the resolved element for an auto: fill step and never touches page.fill', async () => {
				const element = fakeElement();
				vi.mocked(mockPage.evaluateHandle).mockResolvedValue(fakeHandle(element));

				await browserManager.executePreScanActions(mockPage, [
					{ type: 'fill', selector: 'auto:password', value: 'hunter2' }
				]);

				expect(element.fill).toHaveBeenCalledWith('hunter2', { timeout: 30_000 });
				expect(mockPage.fill).not.toHaveBeenCalled();
			});

			it('does not auto-fill a sensitive value when masking fails', async () => {
				const element = fakeElement();
				vi.mocked(mockPage.evaluateHandle).mockResolvedValue(fakeHandle(element));
				vi.mocked(mockPage.addStyleTag).mockRejectedValueOnce(
					new Error('style-src blocks inline styles')
				);
				vi.mocked(mockPage.evaluate).mockRejectedValueOnce(
					new Error('execution context destroyed')
				);

				await expect(
					browserManager.executePreScanActions(
						mockPage,
						[
							{
								type: 'fill',
								selector: 'auto:password',
								value: 'auto-mask-canary-51a8'
							}
						],
						{ maskInputValues: true }
					)
				).rejects.toThrow('Failed to install sensitive-field masking; refusing to continue');
				expect(element.fill).not.toHaveBeenCalled();
				expect(element.dispose).toHaveBeenCalled();
			});

			it('clicks the resolved element for auto:submit', async () => {
				const element = fakeElement();
				vi.mocked(mockPage.evaluateHandle).mockResolvedValue(fakeHandle(element));

				await browserManager.executePreScanActions(mockPage, [
					{ type: 'click', selector: 'auto:submit' }
				]);

				expect(element.click).toHaveBeenCalledWith({ timeout: 30_000 });
				expect(mockPage.click).not.toHaveBeenCalled();
			});

			it('falls back to pressing Enter when auto:submit finds no button', async () => {
				vi.mocked(mockPage.evaluateHandle).mockResolvedValue(fakeHandle(null));

				await browserManager.executePreScanActions(mockPage, [
					{ type: 'click', selector: 'auto:submit', timeout: 0 }
				]);

				expect(mockPage.keyboard.press).toHaveBeenCalledWith('Enter');
			});

			it('throws when an auto: fill field cannot be detected', async () => {
				vi.mocked(mockPage.evaluateHandle).mockResolvedValue(fakeHandle(null));

				await expect(
					browserManager.executePreScanActions(mockPage, [
						{ type: 'fill', selector: 'auto:username', value: 'x', timeout: 0 }
					])
				).rejects.toThrow(/could not find the "username" field/);
			});
		});
	});

	describe('close', () => {
		it('keeps the context guard while page handlers drain and close', async () => {
			const calls: string[] = [];
			let guard: ((route: unknown) => Promise<void>) | undefined;
			const context = {
				route: vi.fn().mockImplementation((_pattern, callback) => {
					guard = callback;
					calls.push('guard');
					return Promise.resolve();
				}),
				unroute: vi.fn().mockImplementation(() => {
					calls.push('unguard');
					return Promise.resolve();
				})
			};
			let closed = false;
			const page = {
				context: () => context,
				isClosed: () => closed,
				unrouteAll: vi.fn().mockImplementation(async (options) => {
					expect(options).toEqual({ behavior: 'wait' });
					calls.push('drain');
					const abort = vi.fn().mockResolvedValue(undefined);
					await guard?.({
						request: () => ({ serviceWorker: () => null, frame: () => ({ page: () => page }) }),
						abort
					});
					expect(abort).toHaveBeenCalledWith('blockedbyclient');
				}),
				close: () => {
					closed = true;
					calls.push('close');
					return Promise.resolve();
				}
			} as unknown as Page;
			vi.mocked(mockBrowser.newContext).mockResolvedValueOnce(context as unknown as BrowserContext);
			await browserManager.createContext();
			await browserManager.closePage(page);
			expect(calls).toEqual(['guard', 'drain', 'close']);
			expect(context.unroute).not.toHaveBeenCalled();
		});

		it('should close browser', async () => {
			await browserManager.launch();
			await browserManager.close();

			expect(mockBrowser.close).toHaveBeenCalled();
			expect(browserManager.getBrowser()).toBeNull();
		});

		it('should do nothing if browser not launched', async () => {
			await browserManager.close();

			expect(mockBrowser.close).not.toHaveBeenCalled();
		});
	});

	describe('getBrowser', () => {
		it('should return null before launch', () => {
			expect(browserManager.getBrowser()).toBeNull();
		});

		it('should return browser after launch', async () => {
			await browserManager.launch();

			expect(browserManager.getBrowser()).toBe(mockBrowser);
		});

		it('should return null after close', async () => {
			await browserManager.launch();
			await browserManager.close();

			expect(browserManager.getBrowser()).toBeNull();
		});
	});

	describe('getConfig', () => {
		it('should return a copy of config', () => {
			const config1 = browserManager.getConfig();
			const config2 = browserManager.getConfig();

			expect(config1).toEqual(config2);
			expect(config1).not.toBe(config2); // Different objects
		});

		it('should not expose internal config to mutation', () => {
			const config = browserManager.getConfig();
			config.headless = false;

			const newConfig = browserManager.getConfig();
			expect(newConfig.headless).toBe(true); // Original unchanged
		});
	});
});
