import type { IssueDetail } from '../types/unified-report';

import { isBoldWeight, isLargeText, parseAxeFontSize } from '../utils/contrast';

export interface AxeContrastData {
	fgColor?: string;
	bgColor?: string;
	contrastRatio?: number | string;
	expectedContrastRatio?: number | string;
	fontSize?: string | number;
	fontWeight?: string | number;
	messageKey?: string;
}

export type SampleSlot = 'fg' | 'bg';

export function isColorContrastIssue(issue: Pick<IssueDetail, 'ruleId'>): boolean {
	return issue.ruleId?.includes('color-contrast') ?? false;
}

export function isAxeIncompleteIssue(issue: Pick<IssueDetail, 'scannerData'>): boolean {
	return issue.scannerData?.axeIncomplete === true;
}

export function getContrastData(issue: Pick<IssueDetail, 'scannerData'>): AxeContrastData | null {
	const raw = issue.scannerData?.contrastData;
	if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return null;
	return raw;
}

export function getDefaultLargeText(data: AxeContrastData | null): boolean | null {
	const px = parseAxeFontSize(data?.fontSize ?? null);
	if (px === null) return null;
	return isLargeText(px, isBoldWeight(data?.fontWeight ?? null));
}
