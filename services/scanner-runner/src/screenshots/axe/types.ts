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

/** Diagnostic info for debugging screenshot coordinate alignment */
export interface PageOverviewDiagnostics {
	animationFinishedCount: number;
	animationPausedCount: number;
	assetWaitStatus: 'completed' | 'failed' | 'skipped' | 'timed_out';
	contentVisibilityElementCount: number;
	contentVisibilityForced: boolean;
	cssPageWidth: number;
	cssPageHeight: number;
	fontWaitStatus: 'completed' | 'failed' | 'not_supported' | 'timed_out';
	lazyMediaCount: number;
	screenshotWidth: number;
	screenshotHeight: number;
	scaleX: number;
	scaleY: number;
	devicePixelRatio: number;
	captureHeight: number;
	elementCount: number;
	preScrollAfterHeight: number;
	preScrollBeforeHeight: number;
	preScrollCompleted: boolean;
	preScrollSteps: number;
	wasClipped: boolean;
	elements: {
		selector: string;
		rawBox: { x: number; y: number; width: number; height: number } | null;
		scaledBox: { x: number; y: number; width: number; height: number } | null;
		percentBounds: {
			x: number;
			y: number;
			width: number;
			height: number;
		} | null;
		skipped: boolean;
		skipReason?: string;
	}[];
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
