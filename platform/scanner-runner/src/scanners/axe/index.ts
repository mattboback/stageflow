import AxeBuilder from "@axe-core/playwright";
import { mkdir } from "node:fs/promises";
import { join } from "node:path";

import { getRuleBehavior } from "../../config/rule-behaviors";
import { getUserImpact } from "../../config/user-impact";
import {
  type Issue,
  type PageScanResult,
  type ScanContext,
  ScannerBase,
  type ScannerMetadata,
} from "../../core";
import { extractContextSnippet } from "../../screenshots/axe/context-snippet";
import {
  AxeScreenshotService,
  type AxeViolation,
  type EnhancedScreenshotResult,
  type ViolationCaptureFailure,
  type PageOverviewViolation,
} from "../../screenshots/AxeScreenshotService";
import { normalizeSeverity } from "../../utils/severity";

const PACKAGE_VERSION = process.env.npm_package_version?.trim() ?? "1.0.0";

/**
 * Default milliseconds to wait after load for dynamic content to render.
 * SPAs and JS-heavy sites often need this extra time for React/Vue/Svelte hydration.
 * Reduced from 500ms to 50ms since most modern frameworks hydrate quickly.
 */
const DEFAULT_DYNAMIC_CONTENT_WAIT_MS = 50;

/**
 * Maximum time to wait for networkidle state before proceeding with scan.
 * Prevents hanging on sites with persistent connections (WebSockets, long-polling).
 */
const NETWORKIDLE_TIMEOUT_MS = 5_000;

interface AxeNode {
  target?: string[];
  html?: string;
  failureSummary?: string;
  contextHtml?: string;
  ancestorPath?: string;
}

interface AxeViolationResult {
  id?: string;
  impact?: string;
  description?: string;
  help?: string;
  helpUrl?: string;
  nodes?: AxeNode[];
  tags?: string[];
}

/**
 * Options for the axe scanner.
 * Pass via SCANNER_OPTIONS environment variable as JSON.
 */
export interface AxeOptions {
  /**
   * Milliseconds to wait after networkidle for dynamic content to render.
   * Increase for heavy SPAs (React, Vue, Svelte) that need more hydration time.
   * Set to 0 to skip the wait entirely.
   * Default: 500
   */
  dynamicContentWaitMs?: number;

  /**
   * axe-core rules to disable. Useful for suppressing known false positives.
   * Example: ["color-contrast", "region"]
   */
  disabledRules?: string[];

  /**
   * WCAG tags to limit the scan to. If not set, all rules run.
   * Example: ["wcag2a", "wcag2aa", "wcag21aa"]
   */
  runOnlyTags?: string[];
}

function parseAxeOptions(raw: unknown): AxeOptions {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    return { dynamicContentWaitMs: DEFAULT_DYNAMIC_CONTENT_WAIT_MS };
  }

  const record = raw as Record<string, unknown>;
  const options: AxeOptions = {};

  // Parse dynamicContentWaitMs
  const waitMs = record.dynamicContentWaitMs;
  if (typeof waitMs === "number" && waitMs >= 0) {
    options.dynamicContentWaitMs = waitMs;
  } else {
    options.dynamicContentWaitMs = DEFAULT_DYNAMIC_CONTENT_WAIT_MS;
  }

  // Parse disabledRules
  const disabledRules = record.disabledRules;
  if (Array.isArray(disabledRules)) {
    options.disabledRules = disabledRules.filter(
      (r): r is string => typeof r === "string" && r.length > 0,
    );
  }

  // Parse runOnlyTags
  const runOnlyTags = record.runOnlyTags;
  if (Array.isArray(runOnlyTags)) {
    options.runOnlyTags = runOnlyTags.filter(
      (t): t is string => typeof t === "string" && t.length > 0,
    );
  }

  return options;
}

export class AxeScanner extends ScannerBase {
  readonly metadata: ScannerMetadata = {
    name: "axe",
    version: PACKAGE_VERSION,
    description: "Accessibility scanner powered by axe-core",
  };

  private screenshotService: AxeScreenshotService;
  private options: AxeOptions = { dynamicContentWaitMs: DEFAULT_DYNAMIC_CONTENT_WAIT_MS };

  constructor() {
    super();
    this.screenshotService = new AxeScreenshotService();
  }

  protected override async initialize(): Promise<void> {
    await super.initialize();
    this.options = parseAxeOptions(this.config.options);
    this.logger.info("Axe options", {
      dynamicContentWaitMs: this.options.dynamicContentWaitMs,
      disabledRules: this.options.disabledRules,
      runOnlyTags: this.options.runOnlyTags,
    });
  }

  async scanPage(context: ScanContext): Promise<PageScanResult> {
    const { page, pageEntry, resultsDir, logger } = context;
    const startedAt = new Date().toISOString();
    const hrStart = process.hrtime.bigint();

    try {
      // Wait for networkidle with timeout to handle sites with persistent connections.
      // If timeout is reached, proceed anyway - the page is likely ready enough for scanning.
      try {
        await Promise.race([
          page.waitForLoadState("networkidle"),
          new Promise<void>((resolve) => setTimeout(resolve, NETWORKIDLE_TIMEOUT_MS)),
        ]);
      } catch {
        // Ignore networkidle timeout - page may still be scannable
        logger.debug("Networkidle wait timed out, proceeding with scan", {
          url: pageEntry.url,
          timeoutMs: NETWORKIDLE_TIMEOUT_MS,
        });
      }

      const waitMs = this.options.dynamicContentWaitMs ?? DEFAULT_DYNAMIC_CONTENT_WAIT_MS;
      if (waitMs > 0) {
        await page.waitForTimeout(waitMs);
      }

      logger.debug("Running axe-core analysis", {
        url: pageEntry.url,
        dynamicContentWaitMs: waitMs,
      });

      let axe = new AxeBuilder({ page });

      // Apply disabled rules if configured
      if (this.options.disabledRules && this.options.disabledRules.length > 0) {
        axe = axe.disableRules(this.options.disabledRules);
      }

      // Apply runOnly tags if configured
      if (this.options.runOnlyTags && this.options.runOnlyTags.length > 0) {
        axe = axe.withTags(this.options.runOnlyTags);
      }

      const axeResults = await axe.analyze();

      logger.info("Axe analysis complete", {
        url: pageEntry.url,
        violations: axeResults.violations.length,
        passes: axeResults.passes.length,
        incomplete: axeResults.incomplete.length,
        inapplicable: axeResults.inapplicable.length,
      });

      if (axeResults.violations.length > 0) {
        logger.info("Axe violations found", {
          violationIds: axeResults.violations.map((v) => ({
            id: v.id,
            impact: v.impact,
            nodes: v.nodes.length,
          })),
        });
      }

      const violations = axeResults.violations as AxeViolationResult[];

      await mkdir(resultsDir, { recursive: true });
      const screenshotsDir = join(resultsDir, "screenshots");
      await mkdir(screenshotsDir, { recursive: true });

      const issues: Issue[] = [];

      // Process violations in parallel batches for better performance
      const BATCH_SIZE = 5;
      const SCREENSHOT_TIMEOUT = 10_000;
      const ENRICHMENT_TIMEOUT = 5_000;

      const processViolation = async (violation: AxeViolationResult) => {
        const [screenshotResult, enrichedNodes] = await Promise.all([
          Promise.race([
            this.captureViolationScreenshot(page, violation as AxeViolation, screenshotsDir),
            new Promise<null>((resolve) => {
              setTimeout(() => { resolve(null); }, SCREENSHOT_TIMEOUT);
            })
          ]),
          Promise.race([
            this.enrichNodesWithContext(page, violation.nodes ?? []),
            new Promise<AxeNode[]>((resolve) => {
              setTimeout(() => { resolve([...(violation.nodes ?? [])]); }, ENRICHMENT_TIMEOUT);
            })
          ])
        ]);
        return { violation, screenshotResult, enrichedNodes };
      };

      // Process in batches to avoid overwhelming the browser
      for (let i = 0; i < violations.length; i += BATCH_SIZE) {
        const batch = violations.slice(i, i + BATCH_SIZE);
        const results = await Promise.allSettled(batch.map(processViolation));

        for (const result of results) {
          if (result.status === 'fulfilled') {
            const { violation, screenshotResult, enrichedNodes } = result.value;
            const issue = this.mapViolationToIssue(violation, screenshotResult, enrichedNodes);
            issues.push(issue);
          }
        }
      }

      // Capture page overview screenshot with all violations highlighted
      const pageOverviewViolations: PageOverviewViolation[] = violations.map((v) => ({
        id: v.id ?? "unknown",
        impact: v.impact,
        nodes: v.nodes,
      }));

	      const pageOverview = await this.screenshotService.capturePageOverview(
	        page,
	        pageOverviewViolations,
	        screenshotsDir,
	        pageEntry.id,
	        { scannerId: this.metadata.name },
	      );

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
          ...axeResults,
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

  private async captureViolationScreenshot(
    page: import("playwright").Page,
    violation: AxeViolation,
    screenshotsDir: string,
  ): Promise<EnhancedScreenshotResult | null> {
    try {
      const outcome = await this.screenshotService.captureViolationScreenshot(
        page,
        violation,
        screenshotsDir,
      );
      if (outcome.status === "captured") {
        this.logScreenshotFallbacks(violation.id, outcome.fallbacks);

        return outcome.screenshot;
      }

      if (outcome.status === "failed") {
        this.logScreenshotFallbacks(violation.id, outcome.fallbacks);
        this.logger.debug("Screenshot capture failed", {
          ruleId: violation.id,
          step: outcome.failure.step,
          reason: outcome.failure.reason,
          message: outcome.failure.message,
        });

        return null;
      }

      this.logger.debug("Screenshot capture skipped", {
        ruleId: violation.id,
        reason: outcome.reason,
      });

      return null;
    } catch {
      this.logger.debug("Screenshot capture threw unexpected error", {
        ruleId: violation.id,
      });

      return null;
    }
  }

  private logScreenshotFallbacks(
    ruleID: string | undefined,
    fallbacks: ViolationCaptureFailure[],
  ): void {
    for (const fallback of fallbacks) {
      this.logger.debug("Screenshot strategy fallback", {
        ruleId: ruleID,
        step: fallback.step,
        reason: fallback.reason,
        message: fallback.message,
      });
    }
  }

  /**
   * Enrich violation nodes with DOM context snippets.
   * Extracts contextHtml and ancestorPath for each node (up to 5).
   */
  private async enrichNodesWithContext(
    page: import("playwright").Page,
    nodes: AxeNode[],
  ): Promise<AxeNode[]> {
    const enrichedNodes: AxeNode[] = [];

    // Only process first 5 nodes (same limit as in mapViolationToIssue)
    const nodesToProcess = nodes.slice(0, 5);

    for (const node of nodesToProcess) {
      const selector = node.target?.[0];
      if (!selector) {
        // No selector to extract context from, keep original node
        enrichedNodes.push({ ...node });
        continue;
      }

      try {
        const contextResult = await extractContextSnippet(page, selector);
        enrichedNodes.push({
          ...node,
          contextHtml: contextResult?.contextHtml,
          ancestorPath: contextResult?.ancestorPath,
        });
      } catch {
        // Failed to extract context, keep original node
        enrichedNodes.push({ ...node });
      }
    }

    return enrichedNodes;
  }

  private mapViolationToIssue(
    violation: AxeViolationResult,
    screenshotResult: EnhancedScreenshotResult | null | undefined,
    enrichedNodes: AxeNode[],
  ): Issue {
    const severity = normalizeSeverity(violation.impact, "info");
    const behavior = getRuleBehavior(violation.id);
    const userImpact = getUserImpact(violation.id);
    const nodes = violation.nodes ?? [];

    const primaryNode = enrichedNodes[0] ?? nodes[0];
    const selector = primaryNode?.target?.[0];

    const category = this.extractCategory(violation.tags);

    return {
      id: violation.id ?? "unknown",
      scanner: this.metadata.name,
      severity,
      category,
      title: violation.help ?? violation.id ?? "Accessibility Issue",
      description: behavior.summary ?? violation.description ?? "",
      helpUrl: violation.helpUrl,
      location: {
        selector,
        html: primaryNode?.html,
      },
      screenshot: screenshotResult?.screenshot,
      metadata: {
        impact: violation.impact,
        tags: violation.tags,
        nodeCount: nodes.length,
        nodes: enrichedNodes.slice(0, 5).map((node) => ({
          target: node.target,
          html: node.html,
          failureSummary: node.failureSummary,
          contextHtml: node.contextHtml,
          ancestorPath: node.ancestorPath,
        })),
        ruleBehavior: behavior,
        friendlyNode: screenshotResult?.friendlyNode,
        locationInfo: screenshotResult?.locationInfo,
        thumbnail: screenshotResult?.thumbnail,
        elementBounds: screenshotResult?.elementBounds,
        // User impact for stakeholder communication
        userImpact: {
          statement: userImpact.statement,
          affectedGroups: userImpact.affectedGroups,
          severity: userImpact.severity,
          userStory: userImpact.userStory,
        },
      },
    };
  }

  private extractCategory(tags?: string[]): string {
    if (!tags || tags.length === 0) {
      return "accessibility";
    }

    const wcagTag = tags.find((t) => t.startsWith("wcag"));
    if (wcagTag) {
      return wcagTag;
    }

    if (tags.includes("best-practice")) {
      return "best-practice";
    }

    const categoryTags = [
      "cat.color",
      "cat.forms",
      "cat.keyboard",
      "cat.language",
      "cat.name-role-value",
      "cat.parsing",
      "cat.semantics",
      "cat.sensory-and-visual-cues",
      "cat.structure",
      "cat.tables",
      "cat.text-alternatives",
      "cat.time-and-media",
    ];

    for (const tag of tags) {
      if (categoryTags.includes(tag)) {
        return tag.replace("cat.", "");
      }
    }

    return "accessibility";
  }
}
