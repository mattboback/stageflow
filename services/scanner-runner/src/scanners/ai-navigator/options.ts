import type { AgentGoal, SuccessCriterion, VisionConfig } from "../../ai";

export interface AiNavigatorOptions {
	goal: AgentGoal;
	vision: Omit<VisionConfig, "apiKey" | "appTitle" | "appReferer">;
}

export function parseAiNavigatorOptions(raw: unknown): AiNavigatorOptions {
	if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
		throw new Error(
			"ai-navigator requires SCANNER_OPTIONS to be a JSON object",
		);
	}

	const record = raw as Record<string, unknown>;
	const goalRaw = record.goal;
	const visionRaw = record.vision;

	const goal = parseAgentGoal(goalRaw);
	const vision = parseVisionConfig(visionRaw);

	return { goal, vision };
}

function parseAgentGoal(raw: unknown): AgentGoal {
	if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
		throw new Error("ai-navigator requires options.goal to be a JSON object");
	}

	const record = raw as Record<string, unknown>;
	const objective = record.objective;
	if (typeof objective !== "string" || !objective.trim()) {
		throw new Error("ai-navigator requires goal.objective (string)");
	}

	const goal: AgentGoal = { objective };

	const maxSteps = record.maxSteps;
	if (
		typeof maxSteps === "number" &&
		Number.isFinite(maxSteps) &&
		maxSteps > 0
	) {
		goal.maxSteps = maxSteps;
	}

	const maxWallTimeMs = record.maxWallTimeMs;
	if (
		typeof maxWallTimeMs === "number" &&
		Number.isFinite(maxWallTimeMs) &&
		maxWallTimeMs > 0
	) {
		goal.maxWallTimeMs = maxWallTimeMs;
	}

	const inputValues = record.inputValues;
	if (
		inputValues &&
		typeof inputValues === "object" &&
		!Array.isArray(inputValues)
	) {
		const values = inputValues as Record<string, unknown>;
		const resolved: Record<string, string> = {};
		for (const [key, value] of Object.entries(values)) {
			if (typeof value === "string") {
				resolved[key] = value;
			}
		}

		if (Object.keys(resolved).length > 0) {
			goal.inputValues = resolved;
		}
	}

	const successCriteria = record.successCriteria;
	if (Array.isArray(successCriteria)) {
		const parsed = successCriteria
			.map((c) => parseSuccessCriterion(c))
			.filter((c): c is NonNullable<typeof c> => Boolean(c));

		if (parsed.length > 0) {
			goal.successCriteria = parsed;
		}
	}

	return goal;
}

function parseSuccessCriterion(raw: unknown): SuccessCriterion | undefined {
	if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
		return undefined;
	}

	const record = raw as Record<string, unknown>;
	const type = record.type;
	const value = record.value;
	if (typeof type !== "string" || typeof value !== "string") {
		return undefined;
	}

	if (
		type !== "url-contains" &&
		type !== "url-matches" &&
		type !== "element-visible" &&
		type !== "text-visible" &&
		type !== "custom"
	) {
		return undefined;
	}

	return { type, value };
}

function parseVisionConfig(raw: unknown): AiNavigatorOptions["vision"] {
	const record = requireObject(
		raw,
		"ai-navigator requires options.vision to be a JSON object",
	);

	rejectOpenRouterAuthHeaderFields(record);

	const model = record.model;
	if (typeof model !== "string" || !model.trim()) {
		throw new Error("ai-navigator requires vision.model (string)");
	}

	const vision: AiNavigatorOptions["vision"] = {
		provider: "openrouter",
		model,
	};

	assignPositiveNumber(vision, "maxTokens", record.maxTokens);
	assignPositiveNumber(vision, "timeoutMs", record.timeoutMs);
	assignPositiveNumber(vision, "maxImageBytes", record.maxImageBytes);
	assignPositiveNumber(vision, "maxConcurrency", record.maxConcurrency);

	const retry = parseRetryConfig(record.retry);
	if (retry) {
		vision.retry = retry;
	}

	return vision;
}

function requireObject(raw: unknown, message: string): Record<string, unknown> {
	if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
		throw new Error(message);
	}

	return raw as Record<string, unknown>;
}

function rejectOpenRouterAuthHeaderFields(
	record: Record<string, unknown>,
): void {
	if (typeof record.apiKey === "string" && record.apiKey.trim()) {
		throw new Error(
			"Do not set vision.apiKey in SCANNER_OPTIONS; use OPENROUTER_API_KEY env var",
		);
	}
	if (typeof record.appTitle === "string" && record.appTitle.trim()) {
		throw new Error(
			"Do not set vision.appTitle in SCANNER_OPTIONS; use OPENROUTER_APP_TITLE env var",
		);
	}
	if (typeof record.appReferer === "string" && record.appReferer.trim()) {
		throw new Error(
			"Do not set vision.appReferer in SCANNER_OPTIONS; use OPENROUTER_APP_REFERER env var",
		);
	}
}

function parsePositiveFiniteNumber(value: unknown): number | undefined {
	if (typeof value !== "number" || !Number.isFinite(value) || value <= 0) {
		return undefined;
	}

	return value;
}

function assignPositiveNumber(
	target: AiNavigatorOptions["vision"],
	key: "maxTokens" | "timeoutMs" | "maxImageBytes" | "maxConcurrency",
	value: unknown,
): void {
	const parsed = parsePositiveFiniteNumber(value);
	if (parsed === undefined) {
		return;
	}

	target[key] = parsed;
}

function parseRetryConfig(
	raw: unknown,
): { maxAttempts: number; baseDelayMs?: number } | undefined {
	if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
		return undefined;
	}

	const record = raw as Record<string, unknown>;
	const maxAttempts = parsePositiveFiniteNumber(record.maxAttempts);
	if (maxAttempts === undefined) {
		return undefined;
	}

	const retry: { maxAttempts: number; baseDelayMs?: number } = { maxAttempts };
	const baseDelayMs = parsePositiveFiniteNumber(record.baseDelayMs);
	if (baseDelayMs !== undefined) {
		retry.baseDelayMs = baseDelayMs;
	}

	return retry;
}
