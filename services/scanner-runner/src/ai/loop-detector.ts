import type { AgentStep } from './types';

export function detectLoop(history: AgentStep[]): boolean {
	if (history.length < 6) {
		return false;
	}

	const recent = history.slice(-6);
	const urlCounts = new Map<string, number>();

	for (const step of recent) {
		urlCounts.set(step.url, (urlCounts.get(step.url) ?? 0) + 1);
	}

	return [...urlCounts.values()].some((count) => count >= 3);
}
