import type { IssueSeverity } from '../../core/types';
import type {
	LighthouseAudit,
	LighthouseCategory,
	LighthouseDetailNode,
	LighthouseIssueNode
} from './types';

// Lighthouse audits score 0..1 (binary pass/fail audits score exactly 0 or 1).
// We deliberately do NOT mint 'critical' from a zero score: a single failing
// audit (e.g. errors-in-console, is-crawlable) re-runs on every page and would
// otherwise manufacture a pile of bogus criticals. A hard fail is reported as
// 'serious' at most; severity here is an ordering hint, not a verdict.
export function mapScoreToSeverity(score: number | null): IssueSeverity {
	if (score === null) {
		return 'info';
	}
	if (score < 0.5) {
		return 'serious';
	}
	if (score < 0.9) {
		return 'moderate';
	}
	return 'minor';
}

export function getAuditCategory(
	auditId: string,
	categories: Record<string, LighthouseCategory>
): string | null {
	for (const [categoryId, category] of Object.entries(categories)) {
		if (category.auditRefs.some((ref) => ref.id === auditId)) {
			return categoryId;
		}
	}
	return null;
}

export function getAuditNodeCount(audit: LighthouseAudit): number {
	const items = audit.details?.items;
	if (!Array.isArray(items)) {
		return 1;
	}

	return Math.max(1, items.length);
}

function firstDefined<T>(values: (T | undefined)[]): T | undefined {
	for (const value of values) {
		if (value !== undefined) {
			return value;
		}
	}
	return undefined;
}

function getTrimmedString(value: unknown): string | undefined {
	if (typeof value !== 'string') {
		return undefined;
	}

	const trimmed = value.trim();
	return trimmed.length > 0 ? trimmed : undefined;
}

function getDetailNode(value: unknown): LighthouseDetailNode | undefined {
	if (!value || typeof value !== 'object') {
		return undefined;
	}

	return value;
}

function extractAuditSelector(input: {
	item: Record<string, unknown>;
	node?: LighthouseDetailNode;
	element?: LighthouseDetailNode;
	source?: LighthouseDetailNode;
	relatedNode?: LighthouseDetailNode;
}): string | undefined {
	return firstDefined([
		getTrimmedString(input.node?.selector),
		getTrimmedString(input.item.selector),
		getTrimmedString(input.element?.selector),
		getTrimmedString(input.source?.selector),
		getTrimmedString(input.relatedNode?.selector)
	]);
}

function extractAuditHtmlSnippet(input: {
	auditDisplayValue: unknown;
	item: Record<string, unknown>;
	node?: LighthouseDetailNode;
	element?: LighthouseDetailNode;
}): string | undefined {
	return firstDefined([
		getTrimmedString(input.node?.snippet),
		getTrimmedString(input.item.snippet),
		getTrimmedString(input.element?.snippet),
		getTrimmedString(input.auditDisplayValue)
	]);
}

function extractAuditAncestorPath(input: {
	item: Record<string, unknown>;
	node?: LighthouseDetailNode;
}): string | undefined {
	return firstDefined([getTrimmedString(input.node?.path), getTrimmedString(input.item.path)]);
}

function extractFailureSummary(input: {
	auditDisplayValue: unknown;
	item: Record<string, unknown>;
}): string | undefined {
	return firstDefined([
		getTrimmedString(input.item.explanation),
		getTrimmedString(input.item.displayValue),
		getTrimmedString(input.auditDisplayValue)
	]);
}

export function extractAuditNodes(audit: LighthouseAudit): LighthouseIssueNode[] {
	const items = audit.details?.items;
	if (!Array.isArray(items) || items.length === 0) {
		return [];
	}

	const nodes: LighthouseIssueNode[] = [];
	const seenSelectors = new Set<string>();

	const rawItems: unknown[] = items;

	for (const rawItem of rawItems) {
		if (!rawItem || typeof rawItem !== 'object') {
			continue;
		}

		const item = rawItem as Record<string, unknown>;

		const node = getDetailNode(item.node);
		const element = getDetailNode(item.element);
		const source = getDetailNode(item.source);
		const relatedNode = getDetailNode(item.relatedNode);
		const auditDisplayValue = audit.displayValue;

		const selector = extractAuditSelector({
			item,
			...(node !== undefined ? { node } : {}),
			...(element !== undefined ? { element } : {}),
			...(source !== undefined ? { source } : {}),
			...(relatedNode !== undefined ? { relatedNode } : {})
		});

		if (!selector) {
			continue;
		}

		if (seenSelectors.has(selector)) {
			continue;
		}
		seenSelectors.add(selector);

		const html = extractAuditHtmlSnippet({
			auditDisplayValue,
			item,
			...(node !== undefined ? { node } : {}),
			...(element !== undefined ? { element } : {})
		});

		const ancestorPath = extractAuditAncestorPath({
			item,
			...(node !== undefined ? { node } : {})
		});
		const failureSummary = extractFailureSummary({
			auditDisplayValue,
			item
		});

		nodes.push({
			selector,
			target: [selector],
			...(html !== undefined ? { html } : {}),
			...(ancestorPath !== undefined ? { ancestorPath } : {}),
			...(failureSummary !== undefined ? { failureSummary } : {})
		});

		if (nodes.length >= 5) {
			break;
		}
	}

	return nodes;
}
