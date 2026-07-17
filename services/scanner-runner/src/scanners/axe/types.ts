export interface AxeCheckResult {
	id?: string;
	data?: Record<string, unknown> | null;
}

export interface AxeNode {
	target?: string[];
	html?: string;
	failureSummary?: string;
	contextHtml?: string;
	ancestorPath?: string;
	any?: AxeCheckResult[];
	all?: AxeCheckResult[];
}

export interface AxeViolationResult {
	id?: string;
	impact?: string;
	description?: string;
	help?: string;
	helpUrl?: string;
	nodes?: AxeNode[];
	tags?: string[];
}
