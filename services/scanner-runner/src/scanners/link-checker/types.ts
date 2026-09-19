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
	/** Same host as the scanned page; set by the scanner, not by the request. */
	isInternal?: boolean;
	/** CSS selector for the originating anchor, carried through for visual evidence. */
	selector?: string;
}
