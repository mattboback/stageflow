import type { Page } from "playwright";

import path from "node:path";
import fs from "fs-extra";

import type {
	ActionDecider,
	AgentGoal,
	AgentResult,
	AgentStep,
	PageAnalyzer,
} from "../../ai";
import type { ScreenshotService } from "../../core/screenshots";
import type { PreScanAction, ScannerLogger } from "../../core/types";

import { checkGoal } from "../../ai/goal-checker";

interface PreScanExecutor {
	executePreScanActions(page: Page, actions: PreScanAction[]): Promise<void>;
}

export async function runAiNavigatorAgent(
	page: Page,
	goal: AgentGoal,
	deps: {
		screenshotsDir: string;
		pageAnalyzer: PageAnalyzer;
		actionDecider: ActionDecider;
		screenshotService: ScreenshotService;
		logger: ScannerLogger;
		preScanExecutor: PreScanExecutor;
	},
): Promise<AgentResult> {
	const startUrl = page.url();
	const startedMs = Date.now();

	const maxSteps = goal.maxSteps ?? 10;
	const maxWallTimeMs = goal.maxWallTimeMs ?? 120_000;

	const steps: AgentStep[] = [];
	let stuckReason: string | undefined;

	for (let stepNumber = 1; stepNumber <= maxSteps; stepNumber += 1) {
		if (Date.now() - startedMs > maxWallTimeMs) {
			stuckReason = "Max wall time exceeded";
			const screenshotKey = await captureStepScreenshot(
				page,
				deps.screenshotsDir,
				deps.screenshotService,
				deps.logger,
				stepNumber,
			);
			steps.push({
				stepNumber,
				url: page.url(),
				action: { type: "stuck", reason: stuckReason },
				reasoning: "Stopping due to maxWallTimeMs budget",
				success: false,
				screenshotKey,
				durationMs: 0,
			});
			break;
		}

		const stepStartedMs = Date.now();
		const perception = await deps.pageAnalyzer.analyze(page, goal);
		const decision = await deps.actionDecider.decide(
			page,
			perception,
			goal,
			steps,
		);

		if (decision.action.type === "done") {
			const screenshotKey = await captureStepScreenshot(
				page,
				deps.screenshotsDir,
				deps.screenshotService,
				deps.logger,
				stepNumber,
			);
			steps.push({
				stepNumber,
				url: page.url(),
				action: decision.action,
				reasoning: decision.reasoning,
				success: true,
				screenshotKey,
				durationMs: Date.now() - stepStartedMs,
			});
			break;
		}

		if (decision.action.type === "stuck") {
			stuckReason = decision.action.reason;
			const screenshotKey = await captureStepScreenshot(
				page,
				deps.screenshotsDir,
				deps.screenshotService,
				deps.logger,
				stepNumber,
			);
			steps.push({
				stepNumber,
				url: page.url(),
				action: decision.action,
				reasoning: decision.reasoning,
				success: false,
				error: stuckReason,
				screenshotKey,
				durationMs: Date.now() - stepStartedMs,
			});
			break;
		}

		const preScanAction = decision.action;
		const screenshotKey = await captureStepScreenshot(
			page,
			deps.screenshotsDir,
			deps.screenshotService,
			deps.logger,
			stepNumber,
			preScanAction.type === "click" ? preScanAction.selector : undefined,
		);

		try {
			await deps.preScanExecutor.executePreScanActions(page, [preScanAction]);
			await page.waitForTimeout(250);

			steps.push({
				stepNumber,
				url: page.url(),
				action: decision.action,
				reasoning: decision.reasoning,
				success: true,
				screenshotKey,
				durationMs: Date.now() - stepStartedMs,
			});
		} catch (err) {
			const message = err instanceof Error ? err.message : String(err);
			stuckReason = message;

			steps.push({
				stepNumber,
				url: page.url(),
				action: decision.action,
				reasoning: decision.reasoning,
				success: false,
				error: message,
				screenshotKey,
				durationMs: Date.now() - stepStartedMs,
			});
			break;
		}
	}

	const goalStatus = await checkGoal(page, goal);
	if (!goalStatus.achieved && !stuckReason) {
		stuckReason = goalStatus.reason;
	}

	return {
		success: goalStatus.achieved,
		goal,
		startUrl,
		finalUrl: page.url(),
		steps,
		totalSteps: steps.length,
		totalDurationMs: Date.now() - startedMs,
		stuckReason,
	};
}

async function captureStepScreenshot(
	page: Page,
	screenshotsDir: string,
	screenshotService: ScreenshotService,
	logger: ScannerLogger,
	stepNumber: number,
	highlightSelector?: string,
): Promise<string | undefined> {
	const filename = `ai-step-${String(stepNumber).padStart(3, "0")}.png`;
	const fullPath = path.join(screenshotsDir, filename);

	try {
		if (highlightSelector) {
			const { buffer } = await screenshotService.captureWithHighlights(
				page,
				[{ selector: highlightSelector }],
				{ format: "png" },
			);
			await fs.writeFile(fullPath, buffer);
			return filename;
		}

		const { buffer } = await screenshotService.captureFullPage(page, {
			format: "png",
		});
		await fs.writeFile(fullPath, buffer);
		return filename;
	} catch (err) {
		logger.warn("Failed to capture ai-navigator screenshot", {
			error: err instanceof Error ? err.message : String(err),
		});
		return undefined;
	}
}
