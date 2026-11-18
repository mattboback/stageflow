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

const PAGE_OVERVIEW_DEFAULTS: PageOverviewConfig = {
	enabled: true,
	finishAnimations: true,
	forceContentVisibility: true,
	maxElements: 50,
	maxHeight: 0,
	maxScrollSteps: 80,
	preScroll: true,
	scrollPauseMs: 75,
	scrollSettleMs: 150,
	scrollStepPx: 800
};

export function loadPageOverviewConfig(
	overrides?: Partial<PageOverviewConfig>
): PageOverviewConfig {
	return { ...PAGE_OVERVIEW_DEFAULTS, ...overrides };
}
