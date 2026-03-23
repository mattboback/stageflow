import type { AxeViolation } from './types';

export function normalizeSelectors(values: unknown[]): string[] {
	return values
		.map((value) => (typeof value === 'string' ? value.trim() : String(value).trim()))
		.filter((value) => value.length > 0);
}

export function extractAxeViolationTargets(violation: AxeViolation): {
	targets: string[];
	selector?: string;
} {
	const nodes = violation.nodes ?? [];

	const seen = new Set<string>();
	const targets: string[] = [];

	for (const node of nodes) {
		const nodeTargets = normalizeSelectors(node.target ?? []);
		for (const t of nodeTargets) {
			if (seen.has(t)) {
				continue;
			}
			seen.add(t);
			targets.push(t);
		}
	}

	return {
		targets,
		...(targets[0] !== undefined ? { selector: targets[0] } : {})
	};
}
