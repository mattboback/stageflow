export interface LaunchedChrome {
	port: number;
	pid?: number;
	kill: () => Promise<void>;
}

// Lighthouse audit result types (simplified to match actual Lighthouse output)
export interface LighthouseAudit {
	id: string;
	title: string;
	description: string;
	score: number | null;
	scoreDisplayMode: string;
	displayValue?: string;
	numericValue?: number;
	numericUnit?: string;
	details?: {
		type: string;
		items?: Record<string, unknown>[];
		headings?: { key: string; label: string }[];
	};
}

export interface LighthouseCategory {
	id: string;
	title: string;
	score: number | null;
	auditRefs: { id: string; weight: number }[];
}

export interface LighthouseResult {
	requestedUrl: string;
	finalUrl: string;
	fetchTime: string;
	// Lighthouse can theoretically return partial results; mark optional for defensive coding
	categories?: Record<string, LighthouseCategory>;
	audits?: Record<string, LighthouseAudit>;
}

export interface LighthouseDetailNode {
	selector?: string;
	path?: string;
	snippet?: string;
}

export interface LighthouseIssueNode {
	target?: string[];
	selector?: string;
	html?: string;
	contextHtml?: string;
	ancestorPath?: string;
	textSnippet?: string;
	failureSummary?: string;
}

export interface LighthouseModule {
	default: (
		url: string,
		flags: Record<string, unknown>,
		config: Record<string, unknown>
	) => Promise<{ lhr?: unknown; report?: unknown } | null | undefined>;
}

export interface LighthouseOptions {
	/**
	 * Lighthouse categories to audit. Defaults to accessibility, best-practices, and seo.
	 * Add "performance" for full audits (warning: adds ~2 minutes per page).
	 */
	categories?: string[];
}

// Default categories: skip performance by default since it's expensive (~2 min overhead).
// Performance can be enabled via SCANNER_OPTIONS: { "categories": ["accessibility", "best-practices", "seo", "performance"] }
export const DEFAULT_LIGHTHOUSE_CATEGORIES = ['accessibility', 'best-practices', 'seo'];
export const VALID_LIGHTHOUSE_CATEGORIES = [
	'accessibility',
	'best-practices',
	'seo',
	'performance'
];

export function parseLighthouseOptions(raw: unknown): LighthouseOptions {
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
		return { categories: DEFAULT_LIGHTHOUSE_CATEGORIES };
	}

	const record = raw as Record<string, unknown>;
	const categories = record.categories;

	if (!Array.isArray(categories)) {
		return { categories: DEFAULT_LIGHTHOUSE_CATEGORIES };
	}

	// Filter to valid categories only
	const validCategories = categories
		.filter((c): c is string => typeof c === 'string')
		.filter((c) => VALID_LIGHTHOUSE_CATEGORIES.includes(c));

	if (validCategories.length === 0) {
		return { categories: DEFAULT_LIGHTHOUSE_CATEGORIES };
	}

	return { categories: validCategories };
}
