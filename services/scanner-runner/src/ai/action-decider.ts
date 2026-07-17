import type { Page } from 'playwright';

import type { ActionDecision, AgentGoal, AgentStep, PagePerception } from './types';
import type { VisionClient } from './vision-client';

import { parseActionDecision } from './action-decision-parser';
import { buildDecisionPrompt } from './decision-prompt';
import { checkGoal } from './goal-checker';
import { detectLoop } from './loop-detector';
import { redactAgentInputValues } from './redaction';

export class ActionDecider {
	constructor(private readonly visionClient: VisionClient) {}

	async decide(
		page: Page,
		perception: PagePerception,
		goal: AgentGoal,
		history: AgentStep[]
	): Promise<ActionDecision> {
		const goalStatus = await checkGoal(page, goal);
		if (goalStatus.achieved) {
			return {
				action: { type: 'done' },
				reasoning: goalStatus.reason,
				confidence: goalStatus.confidence
			};
		}

		if (goal.maxSteps != null && history.length >= goal.maxSteps) {
			return {
				action: { type: 'stuck', reason: 'Max steps reached' },
				reasoning: 'Stopping due to maxSteps budget',
				confidence: 0.95
			};
		}

		if (detectLoop(history)) {
			return {
				action: { type: 'stuck', reason: 'Detected a navigation loop' },
				reasoning: 'Recent steps repeatedly returned to the same URL',
				confidence: 0.9
			};
		}

		const screenshot = await page.screenshot({ type: 'png' });
		const prompt = redactAgentInputValues(buildDecisionPrompt(perception, goal, history), goal);
		const response = await this.visionClient.analyze(screenshot, prompt);

		const decision = parseActionDecision(response.content, goal);
		if (decision) {
			return decision;
		}

		const fallback = perception.suggestedActions?.[0];
		if (fallback) {
			return {
				action: fallback.action,
				reasoning: fallback.reasoning,
				confidence: Math.min(0.7, fallback.confidence)
			};
		}

		return {
			action: { type: 'stuck', reason: 'Could not determine next action' },
			reasoning: 'No valid action returned by model and no suggested actions available',
			confidence: 0.3
		};
	}
}
