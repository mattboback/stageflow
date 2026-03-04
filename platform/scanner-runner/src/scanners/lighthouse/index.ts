import fs from "node:fs";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";
import { chromium } from "playwright";

import {
  type Issue,
  type IssueSeverity,
  type PageScanResult,
  type ScanContext,
  ScannerBase,
  type ScannerMetadata,
} from "../../core";
import { extractContextSnippet } from "../../screenshots/axe/context-snippet";
import {
  AxeScreenshotService,
  type PageOverviewViolation,
} from "../../screenshots/AxeScreenshotService";
import { resolvePlaywrightImageChromiumExecutablePath } from "../../utils/playwright";

const PACKAGE_VERSION = process.env.npm_package_version?.trim() ?? "1.0.0";

interface LaunchedChrome {
  port: number;
  pid?: number;
  kill: () => Promise<void>;
}

// Lighthouse audit result types (simplified to match actual Lighthouse output)
interface LighthouseAudit {
  id: string;
  title: string;
  description: string;
  score: number | null;
  scoreDisplayMode: string;
  displayValue?: string;
  numericValue?: number;
  numericUnit?: string;
  details?: {
    type: string;
    items?: Record<string, unknown>[];
    headings?: { key: string; label: string }[];
  };
}

interface LighthouseCategory {
  id: string;
  title: string;
  score: number | null;
  auditRefs: { id: string; weight: number }[];
}

interface LighthouseResult {
  requestedUrl: string;
  finalUrl: string;
  fetchTime: string;
  // Lighthouse can theoretically return partial results; mark optional for defensive coding
  categories?: Record<string, LighthouseCategory>;
  audits?: Record<string, LighthouseAudit>;
}

interface LighthouseDetailNode {
  selector?: string;
  path?: string;
  snippet?: string;
}

interface LighthouseIssueNode {
  target?: string[];
  selector?: string;
  html?: string;
  contextHtml?: string;
  ancestorPath?: string;
  textSnippet?: string;
  failureSummary?: string;
}

// Default categories: skip performance by default since it's expensive (~2 min overhead).
// Performance can be enabled via SCANNER_OPTIONS: { "categories": ["accessibility", "best-practices", "seo", "performance"] }
const DEFAULT_LIGHTHOUSE_CATEGORIES = ["accessibility", "best-practices", "seo"];
const VALID_LIGHTHOUSE_CATEGORIES = ["accessibility", "best-practices", "seo", "performance"];

export interface LighthouseOptions {
  /**
   * Lighthouse categories to audit. Defaults to accessibility, best-practices, and seo.
   * Add "performance" for full audits (warning: adds ~2 minutes per page).
   */
  categories?: string[];
}

function parseLighthouseOptions(raw: unknown): LighthouseOptions {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return { categories: DEFAULT_LIGHTHOUSE_CATEGORIES };
  }

  const record = raw as Record<string, unknown>;
  const categories = record.categories;

  if (!Array.isArray(categories)) {
    return { categories: DEFAULT_LIGHTHOUSE_CATEGORIES };
  }

  // Filter to valid categories only
  const validCategories = categories
    .filter((c): c is string => typeof c === "string")
    .filter((c) => VALID_LIGHTHOUSE_CATEGORIES.includes(c));

  if (validCategories.length === 0) {
    return { categories: DEFAULT_LIGHTHOUSE_CATEGORIES };
  }

  return { categories: validCategories };
}

export class LighthouseScanner extends ScannerBase {
  readonly metadata: ScannerMetadata = {
    name: "lighthouse",
    version: PACKAGE_VERSION,
    description: "Web quality scanner powered by Google Lighthouse",
  };

  private chrome: LaunchedChrome | null = null;
  private lighthouseQueue: Promise<void> = Promise.resolve();
  private screenshotService: AxeScreenshotService;
  private options: LighthouseOptions = { categories: DEFAULT_LIGHTHOUSE_CATEGORIES };

  constructor() {
    super();
    this.screenshotService = new AxeScreenshotService();
  }

  protected override async initialize(): Promise<void> {
    await super.initialize();
    this.options = parseLighthouseOptions(this.config.options);
    this.logger.info("Lighthouse options", { categories: this.options.categories });
  }

  async scanPage(context: ScanContext): Promise<PageScanResult> {
    const { page, pageEntry, resultsDir } = context;
    const startedAt = new Date().toISOString();
    const hrStart = process.hrtime.bigint();

    try {
      await mkdir(resultsDir, { recursive: true });
      const screenshotsDir = join(resultsDir, "screenshots");
      await mkdir(screenshotsDir, { recursive: true });

      const lhResult = await this.runLighthouse(page, pageEntry.url);

       // Lighthouse runs in its own Chrome instance. The Playwright page may have gone stale
       // while waiting in the serialized queue (especially with concurrency > 1). Re-navigate
       // to ensure we have a fresh, responsive page for context enrichment and screenshots.
       // Use 'domcontentloaded' instead of 'networkidle' for speed - we just need the DOM ready.
       try {
         this.logger.debug("Re-navigating Playwright page after Lighthouse", {
           url: pageEntry.url,
         });

        // Prefer BrowserManager navigation to reuse runtime target validation.
        const browserManager = this.browserManager;
        if (browserManager) {
          await browserManager.navigateToPage(page, pageEntry.url, { type: "domcontentloaded" });
        } else {
          await page.goto(pageEntry.url, {
            waitUntil: "domcontentloaded",
            timeout: 15_000,
          });
        }
       } catch (navError) {
         // If re-navigation fails, log it but continue - we'll try enrichment/screenshots anyway
         this.logger.warn("Failed to re-navigate page after Lighthouse, continuing with stale page", {
           url: pageEntry.url,
           error: navError instanceof Error ? navError.message : String(navError),
         });
       }

      const issues = this.extractIssues(lhResult);

      // Context enrichment and screenshots use Playwright page operations that can hang
      // if the page is unresponsive. Wrap with timeouts as a safety net.
      const enrichmentTimeout = 15_000; // Reduced from 30s
      try {
        await Promise.race([
          this.enrichIssuesWithContext(page, issues),
          new Promise<never>((_, reject) =>
            setTimeout(() => { reject(new Error("Context enrichment timed out")); }, enrichmentTimeout),
          ),
        ]);
      } catch (enrichError) {
        this.logger.warn("Context enrichment failed or timed out, continuing without enrichment", {
          error: enrichError instanceof Error ? enrichError.message : String(enrichError),
        });
      }

      const pageOverviewViolations: PageOverviewViolation[] = issues.map((issue) => ({
        id: issue.id,
        impact: issue.severity,
        nodes: (issue.metadata?.nodes as LighthouseIssueNode[] | undefined)?.map((n) => ({
          target: n.target,
        })),
      }));

      const screenshotTimeout = 30_000; // Reduced from 60s
      let pageOverview: Awaited<ReturnType<typeof this.screenshotService.capturePageOverview>> = null;
      try {
        pageOverview = await Promise.race([
          this.screenshotService.capturePageOverview(
            page,
            pageOverviewViolations,
            screenshotsDir,
            pageEntry.id,
            { scannerId: this.metadata.name },
          ),
          new Promise<never>((_, reject) =>
            setTimeout(() => { reject(new Error("Screenshot capture timed out")); }, screenshotTimeout),
          ),
        ]);
      } catch (screenshotError) {
        this.logger.warn("Screenshot capture failed or timed out, continuing without screenshot", {
          error: screenshotError instanceof Error ? screenshotError.message : String(screenshotError),
        });
      }

      const finishedAt = new Date().toISOString();
      const durationNs = process.hrtime.bigint() - hrStart;
      const durationMs = Number(durationNs) / 1e6;

      return {
        pageId: pageEntry.id,
        url: pageEntry.url,
        path: pageEntry.path,
        success: true,
        issues,
        durationMs: Math.round(durationMs * 100) / 100,
        startedAt,
        finishedAt,
        artifacts: pageOverview ? [pageOverview.screenshotPath] : [],
        rawResults: {
          ...lhResult,
          pageOverview: pageOverview
            ? {
                screenshotFilename: pageOverview.screenshotFilename,
                pageWidth: pageOverview.pageWidth,
                pageHeight: pageOverview.pageHeight,
                elements: pageOverview.elements,
              }
            : null,
        },
      };
    } catch (error) {
      const finishedAt = new Date().toISOString();
      const durationNs = process.hrtime.bigint() - hrStart;
      const durationMs = Number(durationNs) / 1e6;

      return {
        pageId: pageEntry.id,
        url: pageEntry.url,
        path: pageEntry.path,
        success: false,
        issues: [],
        durationMs: Math.round(durationMs * 100) / 100,
        startedAt,
        finishedAt,
        error: error instanceof Error ? error.message : String(error),
      };
    }
  }

  private async runLighthouse(
    _page: import("playwright").Page,
    url: string,
  ): Promise<LighthouseResult> {
    // Dynamic import for lighthouse (ESM module)
    const lighthouseModule = await import("lighthouse");
    const lighthouse = lighthouseModule.default;

    return this.runSerialized(async () => {
      const chrome = await this.ensureChrome();
      const port = chrome.port;

      const categories = this.options.categories ?? DEFAULT_LIGHTHOUSE_CATEGORIES;

      this.logger.info("Running Lighthouse", {
        url,
        port,
        categories,
      });

      // Record diagnostic info to stage log for debugging
      this.scanStageLogger?.recordEvent("lighthouse_start", {
        url,
        port,
        categories,
      });

      const flags = {
        port,
        output: "json" as const,
        onlyCategories: categories,
        // Disable throttling for container environments
        throttling: {
          cpuSlowdownMultiplier: 1,
          rttMs: 0,
          throughputKbps: 0,
        },
        // Allow Lighthouse to reset storage between runs when reusing Chrome.
        disableStorageReset: false,
        formFactor: "desktop" as const,
        screenEmulation: {
          mobile: false,
          width: 1280,
          height: 720,
          deviceScaleFactor: 1,
          disabled: false,
        },
      };

      const config = {
        extends: "lighthouse:default",
        settings: {
          onlyCategories: categories,
          formFactor: "desktop" as const,
          throttling: {
            cpuSlowdownMultiplier: 1,
          },
          screenEmulation: {
            mobile: false,
            width: 1280,
            height: 720,
            deviceScaleFactor: 1,
            disabled: false,
          },
        },
      };

      try {
        const runnerResult = await lighthouse(url, flags, config);

        // Defensive checks: Lighthouse types claim these are always defined,
        // but runtime failures can occur. Keep checks as safety net.
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        if (!runnerResult) {
          this.logger.error("Lighthouse returned null/undefined");
          throw new Error("Lighthouse did not return any result");
        }

        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        if (!runnerResult.lhr) {
          this.logger.error("Lighthouse result missing lhr", {
            hasReport: !!runnerResult.report,
            resultKeys: Object.keys(runnerResult),
          });
          throw new Error("Lighthouse did not return LHR (Lighthouse Report)");
        }

        const lhr = runnerResult.lhr as unknown as LighthouseResult;

        const auditsCount = Object.keys(lhr.audits ?? {}).length;
        const categoriesCount = Object.keys(lhr.categories ?? {}).length;
        const categoryIds = Object.keys(lhr.categories ?? {});

        this.logger.info("Lighthouse completed", {
          finalUrl: lhr.finalUrl,
          requestedUrl: lhr.requestedUrl,
          auditsCount,
          categoriesCount,
          categoryIds,
        });

        // Record to stage log for debugging
        this.scanStageLogger?.recordEvent("lighthouse_completed", {
          finalUrl: lhr.finalUrl,
          requestedUrl: lhr.requestedUrl,
          auditsCount,
          categoriesCount,
          categoryIds,
        });

        // Log a sample of audit results for debugging
        const audits = lhr.audits ?? {};
        const auditKeys = Object.keys(audits);
        if (auditKeys.length > 0) {
          const sampleAudits = auditKeys.slice(0, 5).map((key) => {
            const audit = audits[key];
            return {
              id: audit?.id,
              score: audit?.score,
              scoreDisplayMode: audit?.scoreDisplayMode,
            };
          });
          this.logger.info("Sample audits", { sampleAudits });
          this.scanStageLogger?.recordEvent("lighthouse_sample_audits", { sampleAudits });
        } else {
          this.logger.warn("No audits returned from Lighthouse");
          this.scanStageLogger?.recordEvent("lighthouse_no_audits", {});
        }

        return lhr;
      } catch (error) {
        const errorMessage = error instanceof Error ? error.message : String(error);
        this.logger.error("Lighthouse execution failed", {
          error: errorMessage,
          stack: error instanceof Error ? error.stack : undefined,
        });
        this.scanStageLogger?.recordEvent("lighthouse_error", {
          error: errorMessage,
        });
        throw error;
      }
    });
  }

  protected override async cleanup(): Promise<void> {
    await this.closeChrome();
    await super.cleanup();
  }

  private async runSerialized<T>(fn: () => Promise<T>): Promise<T> {
    const previous = this.lighthouseQueue;
    let release!: () => void;
    const current = new Promise<void>((resolve) => {
      release = resolve;
    });
    this.lighthouseQueue = previous.then(() => current);

    await previous;
    try {
      return await fn();
    } finally {
      release();
    }
  }

  private resolveChromePath(): string {
    const envPath = process.env.LIGHTHOUSE_CHROME_PATH?.trim() ?? process.env.CHROME_PATH?.trim() ?? "";
    if (envPath && fs.existsSync(envPath)) {
      return envPath;
    }

    const playwrightPath = chromium.executablePath();
    if (playwrightPath && fs.existsSync(playwrightPath)) {
      return playwrightPath;
    }

    const fallbackExecutable = resolvePlaywrightImageChromiumExecutablePath();
    if (fallbackExecutable) {
      return fallbackExecutable;
    }

    throw new Error(
      "Unable to locate a Chromium/Chrome executable for Lighthouse. Set LIGHTHOUSE_CHROME_PATH.",
    );
  }

  private async ensureChrome(): Promise<LaunchedChrome> {
    if (this.chrome) {
      return this.chrome;
    }

    const chromeLauncherModule = (await import("chrome-launcher")) as unknown as {
      launch?: (opts: Record<string, unknown>) => Promise<LaunchedChrome>;
      default?: { launch?: (opts: Record<string, unknown>) => Promise<LaunchedChrome> };
    };

    const launch = chromeLauncherModule.launch ?? chromeLauncherModule.default?.launch;
    if (typeof launch !== "function") {
      throw new Error("chrome-launcher launch() not available");
    }

    const chromePath = this.resolveChromePath();

    this.logger.info("Launching Chrome for Lighthouse", { chromePath });
    this.scanStageLogger?.recordEvent("lighthouse_chrome_launch", { chromePath });

    const chrome = await launch({
      chromePath,
      // Use a dedicated, headless Chrome process for Lighthouse.
      chromeFlags: [
        "--headless=new",
        "--no-sandbox",
        "--disable-setuid-sandbox",
        "--disable-dev-shm-usage",
        "--disable-gpu",
        "--no-first-run",
        "--no-default-browser-check",
        "--disable-background-networking",
        "--disable-background-timer-throttling",
        "--disable-renderer-backgrounding",
        "--disable-default-apps",
        "--disable-extensions",
        "--disable-sync",
        "--metrics-recording-only",
        "--mute-audio",
      ],
    });

    this.chrome = chrome;
    this.scanStageLogger?.recordEvent("lighthouse_chrome_ready", {
      port: chrome.port,
      pid: chrome.pid ?? null,
    });

    return chrome;
  }

  private async closeChrome(): Promise<void> {
    if (!this.chrome) {
      return;
    }

    const chrome = this.chrome;
    this.chrome = null;

    try {
      await chrome.kill();
      this.scanStageLogger?.recordEvent("lighthouse_chrome_closed", {
        port: chrome.port,
        pid: chrome.pid ?? null,
      });
    } catch (error) {
      this.logger.warn("Failed to close Lighthouse Chrome", {
        error: error instanceof Error ? error.message : String(error),
      });
    }
  }

  private extractIssues(lhResult: LighthouseResult): Issue[] {
    const issues: Issue[] = [];

    const auditEntries = Object.entries(lhResult.audits ?? {});
    this.logger.info("Lighthouse audit extraction", {
      totalAudits: auditEntries.length,
      categories: Object.keys(lhResult.categories ?? {}),
    });

    if (auditEntries.length === 0) {
      this.logger.warn("Lighthouse returned no audits");
      return issues;
    }

    let skippedPassed = 0;
    let skippedNotApplicable = 0;
    let skippedInformative = 0;
    let skippedManual = 0;

    for (const [auditId, audit] of auditEntries) {
      // Skip passed audits, informational, or not applicable
      if (audit.score === 1) {
        skippedPassed++;
        continue;
      }
      if (audit.score === null || audit.scoreDisplayMode === "notApplicable") {
        skippedNotApplicable++;
        continue;
      }
      if (audit.scoreDisplayMode === "informative") {
        skippedInformative++;
        continue;
      }
      if (audit.scoreDisplayMode === "manual") {
        skippedManual++;
        continue;
      }

      const category = this.getAuditCategory(auditId, lhResult.categories ?? {});

      const severity = this.mapScoreToSeverity(audit.score);

      const nodeCount = this.getAuditNodeCount(audit);
      const nodes = this.extractAuditNodes(audit);

      const issue: Issue = {
        id: auditId,
        scanner: this.metadata.name,
        severity,
        category: category ?? "general",
        title: audit.title,
        description: audit.description,
        helpUrl: this.getHelpUrl(auditId),
        metadata: {
          score: audit.score,
          displayValue: audit.displayValue,
          numericValue: audit.numericValue,
          numericUnit: audit.numericUnit,
          details: audit.details,
          nodeCount,
          nodes: nodes.slice(0, 5),
        },
      };

      issues.push(issue);
    }

    this.logger.info("Lighthouse issue extraction complete", {
      issuesFound: issues.length,
      skippedPassed,
      skippedNotApplicable,
      skippedInformative,
      skippedManual,
    });

    // Record extraction stats to stage log
    this.scanStageLogger?.recordEvent("lighthouse_extraction", {
      issuesFound: issues.length,
      skippedPassed,
      skippedNotApplicable,
      skippedInformative,
      skippedManual,
      totalProcessed: auditEntries.length,
    });

    return issues;
  }

  private getAuditNodeCount(audit: LighthouseAudit): number {
    const items = audit.details?.items;
    if (!Array.isArray(items)) {
      return 1;
    }

    return Math.max(1, items.length);
  }

  private firstDefined<T>(values: (T | undefined)[]): T | undefined {
    for (const value of values) {
      if (value !== undefined) {
        return value;
      }
    }
    return undefined;
  }

  private getTrimmedString(value: unknown): string | undefined {
    if (typeof value !== "string") {
      return undefined;
    }

    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : undefined;
  }

  private getDetailNode(value: unknown): LighthouseDetailNode | undefined {
    if (!value || typeof value !== "object") {
      return undefined;
    }

    return value as LighthouseDetailNode;
  }

  private extractAuditSelector(input: {
    item: Record<string, unknown>;
    node?: LighthouseDetailNode;
    element?: LighthouseDetailNode;
    source?: LighthouseDetailNode;
    relatedNode?: LighthouseDetailNode;
  }): string | undefined {
    return this.firstDefined([
      this.getTrimmedString(input.node?.selector),
      this.getTrimmedString(input.item.selector),
      this.getTrimmedString(input.element?.selector),
      this.getTrimmedString(input.source?.selector),
      this.getTrimmedString(input.relatedNode?.selector),
    ]);
  }

  private extractAuditHtmlSnippet(input: {
    auditDisplayValue: unknown;
    item: Record<string, unknown>;
    node?: LighthouseDetailNode;
    element?: LighthouseDetailNode;
  }): string | undefined {
    return this.firstDefined([
      this.getTrimmedString(input.node?.snippet),
      this.getTrimmedString(input.item.snippet),
      this.getTrimmedString(input.element?.snippet),
      this.getTrimmedString(input.auditDisplayValue),
    ]);
  }

  private extractAuditAncestorPath(input: {
    item: Record<string, unknown>;
    node?: LighthouseDetailNode;
  }): string | undefined {
    return this.firstDefined([
      this.getTrimmedString(input.node?.path),
      this.getTrimmedString(input.item.path),
    ]);
  }

  private extractFailureSummary(input: {
    auditDisplayValue: unknown;
    item: Record<string, unknown>;
  }): string | undefined {
    return this.firstDefined([
      this.getTrimmedString(input.item.explanation),
      this.getTrimmedString(input.item.displayValue),
      this.getTrimmedString(input.auditDisplayValue),
    ]);
  }

  private extractAuditNodes(audit: LighthouseAudit): LighthouseIssueNode[] {
    const items = audit.details?.items;
    if (!Array.isArray(items) || items.length === 0) {
      return [];
    }

    const nodes: LighthouseIssueNode[] = [];
    const seenSelectors = new Set<string>();

    const rawItems: unknown[] = items;

    for (const rawItem of rawItems) {
      if (!rawItem || typeof rawItem !== "object") {
        continue;
      }

      const item = rawItem as Record<string, unknown>;

      const node = this.getDetailNode(item.node);
      const element = this.getDetailNode(item.element);
      const source = this.getDetailNode(item.source);
      const relatedNode = this.getDetailNode(item.relatedNode);
      const auditDisplayValue = audit.displayValue;

      const selector = this.extractAuditSelector({
        item,
        node,
        element,
        source,
        relatedNode,
      });

      if (!selector) {
        continue;
      }

      if (seenSelectors.has(selector)) {
        continue;
      }
      seenSelectors.add(selector);

      const html = this.extractAuditHtmlSnippet({
        auditDisplayValue,
        item,
        node,
        element,
      });

      const ancestorPath = this.extractAuditAncestorPath({ item, node });
      const failureSummary = this.extractFailureSummary({ auditDisplayValue, item });

      nodes.push({
        selector,
        target: [selector],
        html,
        ancestorPath,
        failureSummary,
      });

      if (nodes.length >= 5) {
        break;
      }
    }

    return nodes;
  }

  private async enrichNodesWithContext(
    page: import("playwright").Page,
    nodes: LighthouseIssueNode[],
  ): Promise<LighthouseIssueNode[]> {
    const enriched: LighthouseIssueNode[] = [];

    for (const node of nodes.slice(0, 5)) {
      const selector = node.selector ?? node.target?.[0];
      if (!selector) {
        enriched.push({ ...node });
        continue;
      }

      try {
        const ctx = await extractContextSnippet(page, selector);
        enriched.push({
          ...node,
          contextHtml: ctx?.contextHtml ?? node.contextHtml,
          ancestorPath: ctx?.ancestorPath ?? node.ancestorPath,
        });
      } catch {
        enriched.push({ ...node });
      }
    }

    return enriched;
  }

  private async enrichIssuesWithContext(
    page: import("playwright").Page,
    issues: Issue[],
  ): Promise<void> {
    const BATCH_SIZE = 10;
    const NODE_TIMEOUT = 2_000;

    // Flatten all enrichment tasks
    const tasks: { issue: Issue; node: LighthouseIssueNode }[] = [];
    for (const issue of issues) {
      const metadata = issue.metadata;
      if (!metadata || typeof metadata !== "object" || Array.isArray(metadata)) {
        continue;
      }

      const nodes = metadata.nodes as LighthouseIssueNode[] | undefined;
      if (!Array.isArray(nodes)) {
        continue;
      }

      // Limit to first 5 nodes per issue
      for (const node of nodes.slice(0, 5)) {
        tasks.push({ issue, node });
      }
    }

    // Process in batches
    for (let i = 0; i < tasks.length; i += BATCH_SIZE) {
      const batch = tasks.slice(i, i + BATCH_SIZE);
      await Promise.allSettled(batch.map(async ({ node }) => {
        const selector = node.selector ?? node.target?.[0];
        if (!selector) {
          return;
        }

        try {
          const ctx = await Promise.race([
            extractContextSnippet(page, selector),
            new Promise<undefined>((resolve) => {
              setTimeout(() => { resolve(undefined); }, NODE_TIMEOUT);
            })
          ]);
          if (ctx) {
            node.contextHtml = ctx.contextHtml;
            node.ancestorPath = ctx.ancestorPath;
          }
        } catch { /* ignore individual failures */ }
      }));
    }
  }

  private getAuditCategory(
    auditId: string,
    categories: Record<string, LighthouseCategory>,
  ): string | null {
    for (const [categoryId, category] of Object.entries(categories)) {
      if (category.auditRefs.some((ref) => ref.id === auditId)) {
        return categoryId;
      }
    }
    return null;
  }

  private mapScoreToSeverity(score: number | null): IssueSeverity {
    if (score === null) {
      return "info";
    }
    if (score === 0) {
      return "critical";
    }
    if (score < 0.5) {
      return "serious";
    }
    if (score < 0.9) {
      return "moderate";
    }
    return "minor";
  }

  private getHelpUrl(auditId: string): string {
    const docsBase = "https://developer.chrome.com/docs/lighthouse";
    const accessibilityAudits: Record<string, string> = {
      accesskeys: `${docsBase}/accessibility/accesskeys/`,
      "aria-allowed-attr": `${docsBase}/accessibility/aria-allowed-attr/`,
      "aria-hidden-body": `${docsBase}/accessibility/aria-hidden-body/`,
      "aria-hidden-focus": `${docsBase}/accessibility/aria-hidden-focus/`,
      "aria-required-attr": `${docsBase}/accessibility/aria-required-attr/`,
      "aria-roles": `${docsBase}/accessibility/aria-roles/`,
      "aria-valid-attr-value": `${docsBase}/accessibility/aria-valid-attr-value/`,
      "aria-valid-attr": `${docsBase}/accessibility/aria-valid-attr/`,
      "button-name": `${docsBase}/accessibility/button-name/`,
      bypass: `${docsBase}/accessibility/bypass/`,
      "color-contrast": `${docsBase}/accessibility/color-contrast/`,
      "document-title": `${docsBase}/accessibility/document-title/`,
      "duplicate-id-aria": `${docsBase}/accessibility/duplicate-id-aria/`,
      "form-field-multiple-labels": `${docsBase}/accessibility/form-field-multiple-labels/`,
      "frame-title": `${docsBase}/accessibility/frame-title/`,
      "heading-order": `${docsBase}/accessibility/heading-order/`,
      "html-has-lang": `${docsBase}/accessibility/html-has-lang/`,
      "html-lang-valid": `${docsBase}/accessibility/html-lang-valid/`,
      "image-alt": `${docsBase}/accessibility/image-alt/`,
      "input-image-alt": `${docsBase}/accessibility/input-image-alt/`,
      label: `${docsBase}/accessibility/label/`,
      "link-name": `${docsBase}/accessibility/link-name/`,
      list: `${docsBase}/accessibility/list/`,
      listitem: `${docsBase}/accessibility/listitem/`,
      "meta-viewport": `${docsBase}/accessibility/meta-viewport/`,
      "object-alt": `${docsBase}/accessibility/object-alt/`,
      tabindex: `${docsBase}/accessibility/tabindex/`,
      "td-headers-attr": `${docsBase}/accessibility/td-headers-attr/`,
      "th-has-data-cells": `${docsBase}/accessibility/th-has-data-cells/`,
      "valid-lang": `${docsBase}/accessibility/valid-lang/`,
      "video-caption": `${docsBase}/accessibility/video-caption/`,
    };

    return accessibilityAudits[auditId] ?? `${docsBase}/overview/`;
  }
}
