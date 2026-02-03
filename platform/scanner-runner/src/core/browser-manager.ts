/**
 * Browser Manager
 *
 * Playwright browser management for scanner operations.
 */

import { type Browser, type BrowserContext, chromium, type Page } from "playwright";

import { createLogger } from "../utils/logger";
import { resolvePlaywrightImageChromiumExecutablePath } from "../utils/playwright";
import {
  type BrowserConfig,
  DEFAULT_WAIT_STRATEGY,
  type PreScanAction,
  type ScannerLogger,
  type WaitStrategy,
} from "./types";

const DEFAULT_BROWSER_CONFIG: BrowserConfig = {
  headless: true,
  args: [
    "--no-sandbox",
    "--disable-setuid-sandbox",
    "--disable-dev-shm-usage",
    "--disable-gpu",
  ],
  defaultViewport: { width: 1280, height: 720 },
  deviceScaleFactor: 2,
  defaultTimeout: 30_000,
  pageLoadTimeout: 15_000,
};

/**
 * Manages Playwright browser instances for scanner operations.
 */
export class BrowserManager {
  private browser: Browser | null = null;
  private readonly config: BrowserConfig;
  private readonly logger: ScannerLogger;

  constructor(config?: Partial<BrowserConfig>, logger?: ScannerLogger) {
    this.config = { ...DEFAULT_BROWSER_CONFIG, ...config };
    this.logger = logger ?? createLogger("BrowserManager");
  }

  async launch(): Promise<Browser> {
    if (this.browser) {
      return this.browser;
    }

    const fallbackExecutable = resolvePlaywrightImageChromiumExecutablePath();

    const launchAttempts = [
      { name: "default", opts: {} },
      {
        name: "fallback-chrome",
        opts: fallbackExecutable ? { executablePath: fallbackExecutable } : null,
      },
      {
        name: "single-process",
        opts: { args: [...this.config.args, "--single-process"] },
      },
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

        this.logger.info("Launching browser", {
          attempt: attempt.name,
          headless: this.config.headless,
          args: attempt.opts.args ?? this.config.args,
        });

        this.browser = await chromium.launch({
          headless: this.config.headless,
          args: attemptArgs,
          executablePath: attempt.opts.executablePath as string | undefined,
          chromiumSandbox: false,
          env: {
            ...process.env,
            DBUS_SESSION_BUS_ADDRESS: "disabled",
          },
        });

        return this.browser;
      } catch (err) {
        lastError = err;
        this.logger.warn("Browser launch attempt failed", {
          attempt: attempt.name,
          error: err instanceof Error ? err.message : String(err),
        });
      }
    }

    throw lastError instanceof Error ? lastError : new Error(String(lastError));
  }

  async createContext(viewport?: {
    width: number;
    height: number;
  }): Promise<BrowserContext> {
    const browser = await this.launch();

    const vp = viewport ?? this.config.defaultViewport;

    return browser.newContext({
      viewport: vp,
      deviceScaleFactor: this.config.deviceScaleFactor,
      // We inject small amounts of CSS/JS for scanning + screenshots; some sites have strict CSP that would block it.
      bypassCSP: true,
    });
  }

  async navigateToPage(
    page: Page,
    url: string,
    waitStrategy?: WaitStrategy,
  ): Promise<void> {
    const strategy = waitStrategy ?? DEFAULT_WAIT_STRATEGY;

    this.logger.debug("Navigating to page", { url, waitStrategy: strategy.type });

    const waitUntil = this.getPlaywrightWaitUntil(strategy);

    await page.goto(url, {
      waitUntil,
      timeout: this.config.pageLoadTimeout,
    });

    await this.applyAdditionalWait(page, strategy);
  }

  async executePreScanActions(page: Page, actions: PreScanAction[]): Promise<void> {
    if (actions.length === 0) {
      return;
    }

    this.logger.debug("Executing pre-scan actions", { count: actions.length });

    for (const action of actions) {
      await this.executeAction(page, action);
    }
  }

  private async executeAction(page: Page, action: PreScanAction): Promise<void> {
    const timeout =
      "timeout" in action && typeof action.timeout === "number"
        ? action.timeout
        : this.config.defaultTimeout;

    switch (action.type) {
      case "click":
        await page.click(action.selector, { timeout });
        return;
      case "fill":
        await page.fill(action.selector, action.value, { timeout });
        return;
      case "select":
        await page.selectOption(action.selector, action.value, { timeout });
        return;
      case "hover":
        await page.hover(action.selector, { timeout });
        return;
      case "wait":
        await page.waitForTimeout(action.ms);
        return;
      case "scroll":
        if (action.selector) {
          const element = page.locator(action.selector);
          await element.scrollIntoViewIfNeeded({ timeout });
          return;
        }

        await page.evaluate(
          (scrollY) => {
            window.scrollBy(0, scrollY);
          },
          (action.direction === "up" ? -1 : 1) * (action.pixels ?? 500),
        );
        return;
      case "keyboard":
        await page.keyboard.press(action.key);
        return;
      default:
        this.logger.warn("Unknown action type", { action });
        return;
    }
  }

  private getPlaywrightWaitUntil(
    strategy: WaitStrategy,
  ): "load" | "domcontentloaded" | "networkidle" {
    switch (strategy.type) {
      case "domcontentloaded":
        return "domcontentloaded";
      case "networkidle":
        return "networkidle";
      default:
        return "load";
    }
  }

  private async applyAdditionalWait(page: Page, strategy: WaitStrategy): Promise<void> {
    switch (strategy.type) {
      case "selector":
        await page.waitForSelector(strategy.selector, {
          timeout: strategy.timeout ?? this.config.defaultTimeout,
        });
        return;
      case "timeout":
        await page.waitForTimeout(strategy.ms);
        return;
      default:
        return;
    }
  }

  async close(): Promise<void> {
    if (!this.browser) {
      return;
    }

    this.logger.info("Closing browser");
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
