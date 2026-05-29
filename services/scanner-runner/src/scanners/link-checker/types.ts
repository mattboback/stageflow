/**
 * Link Checker Types
 */

export interface LinkInfo {
	href: string;
	text: string;
	isInternal: boolean;
	element: string;
	lineNumber?: number;
	/** CSS selector locating this link in the live DOM, for visual evidence. */
	selector?: string;
}

export interface LinkCheckResult {
	url: string;
	status: number | null;
	error: string | null;
	redirects: string[];
	responseTime: number;
	/** CSS selector for the originating anchor, carried through for visual evidence. */
	selector?: string;
}
