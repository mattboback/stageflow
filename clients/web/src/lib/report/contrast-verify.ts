import type { IssueDetail } from '$lib/types/unified-report';

import { isBoldWeight, isLargeText, parseAxeFontSize } from '$lib/utils/contrast';

/**
 * Axe's color-contrast check data, forwarded by the scanner as
 * `scannerData.contrastData`. All fields are best-effort.
 */
export interface AxeContrastData {
	fgColor?: string;
	bgColor?: string;
	contrastRatio?: number | string;
	expectedContrastRatio?: number | string;
	fontSize?: string | number;
	fontWeight?: string | number;
	messageKey?: string;
}

/** Which color the sampler is currently picking. */
export type SampleSlot = 'fg' | 'bg';

export function isColorContrastIssue(issue: Pick<IssueDetail, 'ruleId'>): boolean {
	return issue.ruleId?.includes('color-contrast') ?? false;
}

/** True for axe `incomplete` results promoted to "needs manual verification" issues. */
export function isAxeIncompleteIssue(issue: Pick<IssueDetail, 'scannerData'>): boolean {
	return issue.scannerData?.axeIncomplete === true;
}

export function getContrastData(issue: Pick<IssueDetail, 'scannerData'>): AxeContrastData | null {
	const raw = issue.scannerData?.contrastData;
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null;
	return raw as AxeContrastData;
}

/**
 * Derive the WCAG large-text default from axe's measured font size/weight.
 * Returns null when axe didn't report a font size (the user decides).
 */
export function getDefaultLargeText(data: AxeContrastData | null): boolean | null {
	const px = parseAxeFontSize(data?.fontSize ?? null);
	if (px === null) return null;
	return isLargeText(px, isBoldWeight(data?.fontWeight ?? null));
}
