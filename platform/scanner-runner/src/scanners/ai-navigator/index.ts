import fs from "fs-extra";
import path from "node:path";

import type { Issue, PageScanResult, ScanContext } from "../../core/types";

import {
  ActionDecider,
  type AgentResult,
  PageAnalyzer,
  VisionClient,
  type VisionConfig,
} from "../../ai";
import { ScannerBase } from "../../core/scanner-base";
import { createScreenshotService, type ScreenshotService } from "../../core/screenshots";
import { runAiNavigatorAgent } from "./agent";
import { type AiNavigatorOptions, parseAiNavigatorOptions } from "./options";
import { uploadAiNavigatorTraces } from "./trace-uploader";

const SCANNER_VERSION = "1.0.0";

export class AiNavigatorScanner extends ScannerBase {
  readonly metadata = {
    name: "ai-navigator",
    version: SCANNER_VERSION,
    description: "AI-powered navigation agent using vision models",
  };

  private screenshotService!: ScreenshotService;
  private visionClient!: VisionClient;
  private pageAnalyzer!: PageAnalyzer;
  private actionDecider!: ActionDecider;
  private options!: AiNavigatorOptions;

  protected override async initialize(): Promise<void> {
    await super.initialize();

    const options = parseAiNavigatorOptions(this.config.options);
    this.options = options;

    const apiKey = process.env.OPENROUTER_API_KEY;
    if (!apiKey) {
      throw new Error("OPENROUTER_API_KEY must be set for ai-navigator");
    }

    const appTitle = process.env.OPENROUTER_APP_TITLE?.trim();
    const appReferer = process.env.OPENROUTER_APP_REFERER?.trim();

    const visionConfig: VisionConfig = {
      ...options.vision,
      provider: "openrouter",
      apiKey,
      appTitle: appTitle ?? undefined,
      appReferer: appReferer ?? undefined,
    };

    this.visionClient = new VisionClient(visionConfig);
    this.pageAnalyzer = new PageAnalyzer(this.visionClient);
    this.actionDecider = new ActionDecider(this.visionClient);
    this.screenshotService = createScreenshotService(this.logger);
  }

  async scanPage(context: ScanContext): Promise<PageScanResult> {
    const startedAt = new Date().toISOString();
    const startedMs = Date.now();

    const tracePath = path.join(context.resultsDir, "ai-trace.json");
    const screenshotsDir = path.join(context.resultsDir, "screenshots");
    await fs.ensureDir(screenshotsDir);

    const goal = this.options.goal;

    const browserManager = this.browserManager;
    if (!browserManager) {
      throw new Error("Browser manager not initialized");
    }

    const agentResult = await runAiNavigatorAgent(context.page, goal, {
      screenshotsDir,
      pageAnalyzer: this.pageAnalyzer,
      actionDecider: this.actionDecider,
      screenshotService: this.screenshotService,
      logger: this.logger,
      preScanExecutor: browserManager,
    });

    await fs.writeJSON(tracePath, agentResult, { spaces: 2 });

    const finishedAt = new Date().toISOString();
    const durationMs = Date.now() - startedMs;

    const issues = agentResult.success ? [] : [this.buildFailureIssue(agentResult)];

    return {
      pageId: context.pageEntry.id,
      url: context.page.url(),
      path: context.pageEntry.path,
      success: agentResult.success,
      issues,
      durationMs,
      startedAt,
      finishedAt,
      error: agentResult.success
        ? undefined
        : (agentResult.stuckReason ?? "Goal not achieved"),
      rawResults: agentResult,
    };
  }

  protected override async uploadArtifacts(): Promise<void> {
    await super.uploadArtifacts();

    const bucket = this.config.storage.bucket;
    const prefix = `${this.config.jobId}/${this.metadata.name}`;

    await uploadAiNavigatorTraces({
      storageProvider: this.storageProvider,
      bucket,
      prefix,
      resultsDir: this.config.resultsDir,
      logger: this.logger,
    });
  }

  private buildFailureIssue(result: AgentResult): Issue {
    return {
      id: "flow-goal-not-achieved",
      scanner: this.metadata.name,
      severity: "critical",
      category: "flow",
      title: "Flow goal not achieved",
      description: result.stuckReason ?? "Agent reported failure",
      metadata: {
        objective: result.goal.objective,
        totalSteps: result.totalSteps,
        startUrl: result.startUrl,
        finalUrl: result.finalUrl,
      },
    };
  }
}
