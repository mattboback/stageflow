import type { AgentGoal } from './types';

import { redactSecretValues, redactStringValues } from '../utils/secret-redaction';

/**
 * Remove execution-only agent input values from text that can leave the
 * browser process (model prompts, traces, reports, and lifecycle events).
 */
export function redactAgentInputValues(value: string, goal?: AgentGoal): string {
	return redactSecretValues(value, Object.values(goal?.inputValues ?? {}));
}

export function redactAgentInputValuesInObject(value: unknown, goal: AgentGoal): unknown {
	return redactStringValues(value, (input) => redactAgentInputValues(input, goal));
}
