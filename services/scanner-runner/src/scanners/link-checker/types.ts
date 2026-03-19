/**
 * Link Checker Types
 */

export interface LinkInfo {
	href: string;
	text: string;
	isInternal: boolean;
	element: string;
	lineNumber?: number;
}

export interface LinkCheckResult {
	url: string;
	status: number | null;
	error: string | null;
	redirects: string[];
	responseTime: number;
}
