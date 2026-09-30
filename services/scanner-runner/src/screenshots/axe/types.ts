import type { IssueSeverity } from '../../core/types';

/** Element overlay with severity for color-coded highlights */
export interface PageOverviewElement {
	issueId: string;
	ruleId: string;
	severity: IssueSeverity;
	selector: string;
	nodeIndex: number;
	// Percentage-based bounds for scale-independent frontend overlays
	xPercent: number;
	yPercent: number;
	widthPercent: number;
	heightPercent: number;
	// Pixel bounds relative to screenshot
	x: number;
	y: number;
	width: number;
	height: number;
}

/** Result of capturing a full-page overview with all issues highlighted */
export interface PageOverviewResult {
	screenshotPath: string;
	screenshotFilename: string;
	pageWidth: number;
	pageHeight: number;
	elements: PageOverviewElement[];
}

/** Input for page overview capture - violation with severity info */
export interface PageOverviewViolation {
	id: string;
	impact?: string;
	nodes?: { target?: string[] }[];
}

export type OutputFormat = 'png' | 'webp';

export interface AxeScreenshotConfig {
	screenshotsEnabled: boolean;
	outputFormat: OutputFormat;
	webpQuality: number;
}
