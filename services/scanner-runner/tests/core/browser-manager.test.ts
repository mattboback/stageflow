/**
 * Browser Manager Tests
 *
 * Comprehensive tests for Playwright browser management.
 */

import fs from 'node:fs';
import { type Browser, type BrowserContext, type Page, chromium } from 'playwright';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import type { PreScanAction, WaitStrategy } from '../../src/core/types';

import { BrowserManager } from '../../src/core/browser-manager';
import { validateRuntimeTargetURL } from '../../src/core/target-validation';

// Mock Playwright and fs
vi.mock('playwright', () => ({
	chromium: {
		launch: vi.fn()
	}
}));

vi.mock('node:fs', () => ({
	default: {
		existsSync: vi.fn(),
		readdirSync: vi.fn()
	}
}));

vi.mock('../../src/core/target-validation', () => ({
	shouldEnforceRuntimeTargetValidation: vi.fn().mockReturnValue(true),
	validateRuntimeTargetURL: vi.fn().mockResolvedValue(undefined)
}));

describe('BrowserManager', () => {
	let browserManager: BrowserManager;
	let mockBrowser: Browser;
	let mockContext: BrowserContext;
	let mockPage: Page;

	beforeEach(() => {
		// Setup mock browser
		mockPage = {
			goto: vi.fn().mockResolvedValue(undefined),
			route: vi.fn().mockResolvedValue(undefined),
			url: vi.fn().mockReturnValue(''),
			click: vi.fn().mockResolvedValue(undefined),
			fill: vi.fn().mockResolvedValue(undefined),
			selectOption: vi.fn().mockResolvedValue(undefined),
			hover: vi.fn().mockResolvedValue(undefined),
			waitForTimeout: vi.fn().mockResolvedValue(undefined),
			waitForSelector: vi.fn().mockResolvedValue(undefined),
			locator: vi.fn().mockReturnValue({
				scrollIntoViewIfNeeded: vi.fn().mockResolvedValue(undefined)
			}),
			evaluate: vi.fn().mockResolvedValue(undefined),
			keyboard: {
				press: vi.fn().mockResolvedValue(undefined)
			}
		} as unknown as Page;

		mockContext = {
			newPage: vi.fn().mockResolvedValue(mockPage),
			close: vi.fn().mockResolvedValue(undefined)
		} as unknown as BrowserContext;

		mockBrowser = {
			newContext: vi.fn().mockResolvedValue(mockContext),
			close: vi.fn().mockResolvedValue(undefined)
		} as unknown as Browser;

		vi.mocked(chromium.launch).mockResolvedValue(mockBrowser);
		vi.mocked(fs.existsSync).mockReturnValue(false);
		vi.mocked(fs.readdirSync).mockReturnValue([]);

		browserManager = new BrowserManager();
	});

	afterEach(() => {
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

		it('should return existing browser if already launched', async () => {
			const browser1 = await browserManager.launch();
			const browser2 = await browserManager.launch();

			expect(browser1).toBe(browser2);
			expect(chromium.launch).toHaveBeenCalledTimes(1);
		});

		it('should attempt fallback chrome executable if exists', async () => {
			vi.mocked(fs.existsSync).mockReturnValue(true);
			vi.mocked(fs.readdirSync).mockReturnValue(['chromium-1200']);
			vi.mocked(chromium.launch).mockRejectedValueOnce(new Error('Launch failed'));

			const browser = await browserManager.launch();

			expect(browser).toBe(mockBrowser);
			expect(chromium.launch).toHaveBeenCalledTimes(2);
			expect(chromium.launch).toHaveBeenNthCalledWith(
				2,
				expect.objectContaining({
					executablePath: '/ms-playwright/chromium-1200/chrome-linux/chrome'
				})
			);
		});

		it('should try single-process mode as final fallback', async () => {
			vi.mocked(fs.existsSync).mockReturnValue(true);
			vi.mocked(fs.readdirSync).mockReturnValue(['chromium-1200']);
			vi.mocked(chromium.launch).mockRejectedValueOnce(new Error('Default failed'));
			vi.mocked(chromium.launch).mockRejectedValueOnce(new Error('Fallback failed'));

			const browser = await browserManager.launch();

			expect(browser).toBe(mockBrowser);
			expect(chromium.launch).toHaveBeenCalledTimes(3);
			expect(chromium.launch).toHaveBeenNthCalledWith(
				3,
				expect.objectContaining({
					args: expect.arrayContaining(['--single-process'])
				})
			);
		});

		it('should throw error if all launch attempts fail', async () => {
			vi.mocked(chromium.launch).mockRejectedValue(new Error('All attempts failed'));

			await expect(browserManager.launch()).rejects.toThrow('All attempts failed');
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

			expect(validateRuntimeTargetURL).toHaveBeenCalledWith('https://example.com');
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

		it('should fail before navigation when target validation fails', async () => {
			vi.mocked(validateRuntimeTargetURL).mockRejectedValueOnce(new Error('Blocked target URL'));

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
			vi.mocked(validateRuntimeTargetURL).mockResolvedValue(undefined);

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

			expect(validateRuntimeTargetURL).toHaveBeenCalledWith('https://example.com/redirected');
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

			vi.mocked(validateRuntimeTargetURL).mockResolvedValue(undefined);

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

			expect(validateRuntimeTargetURL).toHaveBeenCalledWith('https://cdn.example.com/logo.png');
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

			vi.mocked(validateRuntimeTargetURL).mockResolvedValue(undefined);

			const page = {
				...mockPage,
				route: mockRoute,
				url: vi.fn().mockReturnValue('https://example.com')
			} as unknown as Page;

			await expect(
				browserManager.navigateToPage(page, 'https://example.com')
			).resolves.toBeUndefined();

			expect(routeHandler).toBeDefined();

			vi.mocked(validateRuntimeTargetURL).mockRejectedValueOnce(new Error('Blocked subresource'));

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

			// First call is for the initial explicit validateRuntimeTargetURL(url) in navigateToPage.
			// We'll trigger the blocked redirect via the route handler after that completes.
			vi.mocked(validateRuntimeTargetURL).mockResolvedValueOnce(undefined);

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
			vi.mocked(validateRuntimeTargetURL).mockRejectedValueOnce(new Error('Blocked redirect'));

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
	});

	describe('close', () => {
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
