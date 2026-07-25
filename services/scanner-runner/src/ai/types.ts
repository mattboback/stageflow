import type { ActionFill, ActionSelect, PreScanAction } from '../core/types';

export interface VisionConfig {
	provider: 'openrouter';
	apiKey: string;
	model: string;
	appReferer?: string;
	appTitle?: string;
	maxTokens?: number;
	timeoutMs?: number;
	retry?: { maxAttempts: number; baseDelayMs?: number };
	maxImageBytes?: number;
	maxConcurrency?: number;
}

export interface VisionMessage {
	role: 'user' | 'assistant' | 'system';
	content: VisionContent[];
}

export type VisionContent =
	{ type: 'text'; text: string } | { type: 'image_url'; image_url: { url: string } };

export interface VisionResponse {
	content: string;
	usage?: {
		promptTokens: number;
		completionTokens: number;
		totalTokens: number;
	};
}

export interface InteractiveElement {
	selector: string;
	tagName: string;
	role?: string;
	accessibleName?: string;
	text?: string;
	isVisible: boolean;
	isEnabled: boolean;
	boundingBox?: { x: number; y: number; width: number; height: number };
}

export interface PagePerception {
	url: string;
	title: string;
	pageType: string;
	interactiveElements: InteractiveElement[];
	description: string;
	goalRelevance?: number;
	suggestedActions?: SuggestedAction[];
}

export interface SuggestedAction {
	action: PreScanAction;
	reasoning: string;
	confidence: number;
}

export interface AgentGoal {
	objective: string;
	successCriteria?: SuccessCriterion[];
	maxSteps?: number;
	maxCalls?: number;
	maxTokensTotal?: number;
	maxWallTimeMs?: number;
	inputValues?: Record<string, string>;
	/**
	 * Stop after this many consecutive failed action attempts. Ported from the
	 * legacy agent-harness turn loop. Defaults to 3 when unset.
	 */
	maxConsecutiveFailures?: number;
	/**
	 * Stop when this many successive successful turns produced no observable
	 * URL or DOM signature change. Ported from the legacy agent-harness turn
	 * loop. Defaults to 3 when unset.
	 */
	maxNoProgressTurns?: number;
}

export type PublicAgentGoal = Omit<AgentGoal, 'inputValues'> & {
	inputValueKeys?: string[];
};

export interface SuccessCriterion {
	type: 'url-contains' | 'url-matches' | 'element-visible' | 'text-visible' | 'custom';
	value: string;
}

export interface GoalStatus {
	achieved: boolean;
	confidence: number;
	reason: string;
}

export type ResolvedAgentInputAction = (ActionFill | ActionSelect) & {
	value: string;
	valueKey: string;
};

export type AgentAction =
	PreScanAction | ResolvedAgentInputAction | { type: 'done' } | { type: 'stuck'; reason: string };

type NonInputAgentAction = Exclude<AgentAction, { type: 'fill' | 'select' }>;

export type RecordedAgentAction =
	| NonInputAgentAction
	| {
			type: 'fill' | 'select';
			selector: string;
			value: '[REDACTED]';
			valueKey?: string;
			timeout?: number;
	  };

export interface ActionDecision {
	action: AgentAction;
	reasoning: string;
	confidence: number;
	alternatives?: PreScanAction[];
}

export interface AgentStep {
	stepNumber: number;
	url: string;
	action: RecordedAgentAction;
	reasoning: string;
	success: boolean;
	error?: string;
	screenshotKey?: string;
	durationMs: number;
}

export interface AgentResult {
	success: boolean;
	goal: PublicAgentGoal;
	startUrl: string;
	finalUrl: string;
	steps: AgentStep[];
	totalSteps: number;
	totalDurationMs: number;
	stuckReason?: string;
}
