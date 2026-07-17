import { getEnvBool, getEnvInt } from '../../utils/env';

export interface PageOverviewConfig {
	enabled: boolean;
	finishAnimations: boolean;
	forceContentVisibility: boolean;
	maxElements: number;
	maxHeight: number;
	maxScrollSteps: number;
	preScroll: boolean;
	scrollPauseMs: number;
	scrollSettleMs: number;
	scrollStepPx: number;
	skipLargeElements?: boolean;
	largeElementWidthRatio?: number;
	largeElementHeightRatio?: number;
}

export type PageOverviewConfigInput = Partial<PageOverviewConfig> &
	Pick<PageOverviewConfig, 'enabled' | 'maxElements' | 'maxHeight'>;

export function loadPageOverviewConfig(
	overrides?: Partial<{
		enabled: boolean;
		finishAnimations: boolean;
		forceContentVisibility: boolean;
		maxElements: number;
		maxHeight: number;
		maxScrollSteps: number;
		preScroll: boolean;
		scrollPauseMs: number;
		scrollSettleMs: number;
		scrollStepPx: number;
	}>
): {
	enabled: boolean;
	finishAnimations: boolean;
	forceContentVisibility: boolean;
	maxElements: number;
	maxHeight: number;
	maxScrollSteps: number;
	preScroll: boolean;
	scrollPauseMs: number;
	scrollSettleMs: number;
	scrollStepPx: number;
} {
	return {
		enabled: getEnvBool('A11Y_PAGE_OVERVIEW_ENABLED', true),
		finishAnimations: getEnvBool('A11Y_PAGE_OVERVIEW_FINISH_ANIMATIONS', true),
		forceContentVisibility: getEnvBool('A11Y_PAGE_OVERVIEW_FORCE_CONTENT_VISIBILITY', true),
		maxElements: Math.max(0, getEnvInt('A11Y_PAGE_OVERVIEW_MAX_ELEMENTS', 50)),
		maxHeight: Math.max(0, getEnvInt('A11Y_PAGE_OVERVIEW_MAX_HEIGHT', 0)),
		maxScrollSteps: Math.max(1, getEnvInt('A11Y_PAGE_OVERVIEW_MAX_SCROLL_STEPS', 80)),
		preScroll: getEnvBool('A11Y_PAGE_OVERVIEW_PRE_SCROLL', true),
		scrollPauseMs: Math.max(0, getEnvInt('A11Y_PAGE_OVERVIEW_SCROLL_PAUSE_MS', 75)),
		scrollSettleMs: Math.max(0, getEnvInt('A11Y_PAGE_OVERVIEW_SCROLL_SETTLE_MS', 150)),
		scrollStepPx: Math.max(1, getEnvInt('A11Y_PAGE_OVERVIEW_SCROLL_STEP_PX', 800)),
		...overrides
	};
}
