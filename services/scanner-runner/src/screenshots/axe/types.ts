import type { IssueSeverity } from '../../core/types';

export interface AxeViolation {
	id?: string;
	nodes?: { target?: string[] }[];
	screenshot_path?: string;
	thumbnail_path?: string;
}

export interface PageLocationInfo {
	scrollY: number;
	viewportHeight: number;
	docHeight: number;
	position: number; // 0..1 representing approximate position on page
}

export interface FriendlyNodeInfo {
	label: string; // "Button labeled 'Submit'"
	tagName: string; // "button"
	role?: string; // "button"
	name?: string; // accessible name text
	region?: string; // "Main content", "Header", etc.
	textSnippet?: string;
	selector?: string;
}

export interface ElementBounds {
	x: number;
	y: number;
	width: number;
	height: number;
	selector: string;
}

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

export interface EnhancedScreenshotResult {
	screenshot: string;
	thumbnail: string | null;
	locationInfo?: PageLocationInfo;
	friendlyNode?: FriendlyNodeInfo;
	elementBounds?: ElementBounds[]; // Bounding boxes relative to screenshot for overlay
}

export type ViolationCaptureStep = 'semantic' | 'union' | 'single-target' | 'element' | 'viewport';

export type ViolationCaptureFailureReason =
	| 'scroll_failed'
	| 'bounding_box_missing'
	| 'clip_invalid'
	| 'screenshot_failed'
	| 'overlay_failed';

export interface ViolationCaptureFailure {
	step: ViolationCaptureStep;
	reason: ViolationCaptureFailureReason;
	message?: string;
}

export type ViolationScreenshotCaptureResult =
	| {
			status: 'captured';
			screenshot: EnhancedScreenshotResult;
			fallbacks: ViolationCaptureFailure[];
	  }
	| {
			status: 'skipped';
			reason: 'disabled' | 'policy_never';
	  }
	| {
			status: 'failed';
			failure: ViolationCaptureFailure;
			fallbacks: ViolationCaptureFailure[];
	  };

export type HighlightStyle = 'solid' | 'dashed';
export type OutputFormat = 'png' | 'webp';
export type OverlayMethod = 'css-injection' | 'sharp-composite';

export interface AxeScreenshotConfig {
	screenshotsEnabled: boolean;
	injectHighlight: boolean;
	shotScale: number;
	shotMinWidth: number;
	shotMinHeight: number;
	deviceScaleFactor: number;
	highlightStyle: HighlightStyle;
	mergeTargets: boolean;
	unionMaxTargets: number;
	clipPadding: number;
	contextRatio: number;
	/**
	 * How much context to include around a single element when using the
	 * element-only fallback screenshot mode.
	 *
	 * 2.5 means capture a rectangle ~2.5x element size (centered).
	 */
	elementContextMultiplier: number;
	/**
	 * Minimum width/height (in CSS px) for element-only screenshots.
	 * Helps avoid tiny crops for small elements (icons, links, etc.).
	 */
	elementContextMinSize: number;
	scrollTimeout: number;
	thumbnailSize: number;
	generateThumbnails: boolean;
	outputFormat: OutputFormat;
	webpQuality: number;
	overlayMethod: OverlayMethod;
	semanticOverlayEnabled: boolean;
	semanticOverlayMaxHeadings: number;
	semanticOverlayMaxLabelLength: number;
	semanticOverlayLegendEnabled: boolean;
}
