import { useMemo, useState } from 'react';

import type { IssueDetail, PageSummary } from '../../../lib/types/unified-report';
import {
	type SampleSlot,
	findOverviewElement,
	getContrastData,
	getCroppedViewBox,
	getDefaultLargeText,
	isAxeIncompleteIssue
} from '../../../lib/report';
import { describeMessageKey, formatRatio } from '../../../lib/utils/contrast';
import { useContrastVerdicts } from '../../../lib/hooks/useContrastVerdicts';

import { ContrastResult } from './ContrastResult';
import { ContrastSampler } from './ContrastSampler';

interface Props {
	issue: IssueDetail;
	page: PageSummary | null;
	pageOverviewUrl: string | null;
	jobId: string;
}

interface Draft {
	issueId: string;
	fg: string;
	bg: string;
	largeText: boolean;
}

export function VerifyContrastTab({ issue, page, pageOverviewUrl, jobId }: Props) {
	const contrastData = getContrastData(issue);
	const isIncomplete = isAxeIncompleteIssue(issue);
	const { getVerdict, setVerdict, clearVerdict } = useContrastVerdicts(jobId);
	const verdict = getVerdict(issue.id);

	// Draft edits are keyed to the issue; navigating to another issue falls
	// back to that issue's axe prefill instead of resetting in an effect.
	const [draft, setDraft] = useState<Draft | null>(null);
	const state: Draft =
		draft?.issueId === issue.id
			? draft
			: {
					issueId: issue.id,
					fg: contrastData?.fgColor ?? '',
					bg: contrastData?.bgColor ?? '',
					largeText: getDefaultLargeText(contrastData) ?? false
				};
	const setState = (update: (prev: Draft) => Draft) => {
		setDraft(update(state));
	};

	const overviewElement = findOverviewElement(page, issue.id);
	const overview = page?.pageOverview;

	// Slightly roomier than the evidence crop so the eyedropper has background
	// around the element to sample.
	const cropViewBox = useMemo(() => {
		if (!overview || !overviewElement) return null;
		return getCroppedViewBox(overview.pageWidth, overview.pageHeight, overviewElement, {
			padding: 72,
			minWidth: 360,
			minHeight: 220
		});
	}, [overview, overviewElement]);

	const samplerAvailable = Boolean(pageOverviewUrl && cropViewBox && overview);

	const measuredNote = useMemo(() => {
		if (!contrastData) return null;
		const parts: string[] = [];
		if (contrastData.fgColor) parts.push(`text ${contrastData.fgColor}`);
		if (contrastData.bgColor) parts.push(`on ${contrastData.bgColor}`);
		const numericRatio = Number(contrastData.contrastRatio);
		if (Number.isFinite(numericRatio) && numericRatio > 0) {
			parts.push(`· ${formatRatio(numericRatio)}:1`);
		}
		return parts.length > 0 ? parts.join(' ') : null;
	}, [contrastData]);

	const handlePick = (slot: SampleSlot, hex: string) => {
		setState((prev) => (slot === 'fg' ? { ...prev, fg: hex } : { ...prev, bg: hex }));
	};

	return (
		<div className="vfy">
			{isIncomplete ? (
				<div className="vfy__why">
					<p className="vfy__why-head">Why this needs a human</p>
					<p className="vfy__why-body">{describeMessageKey(contrastData?.messageKey)}</p>
					{measuredNote && (
						<p className="vfy__why-note">
							axe measured <span className="mono">{measuredNote}</span> before giving up.
						</p>
					)}
				</div>
			) : (
				measuredNote && (
					<p className="vfy__measured">
						axe measured <span className="mono">{measuredNote}</span> — sample the
						screenshot to double-check it.
					</p>
				)
			)}

			{samplerAvailable && pageOverviewUrl && cropViewBox && overview ? (
				<ContrastSampler
					imageUrl={pageOverviewUrl}
					pageWidth={overview.pageWidth}
					pageHeight={overview.pageHeight}
					viewBox={cropViewBox}
					element={overviewElement}
					onPick={handlePick}
				/>
			) : (
				<p className="vfy__notice">
					No screenshot is available for this element — screenshots may have been
					disabled for this scan. Enter the colors manually below, or re-run the scan
					with screenshots enabled.
				</p>
			)}

			<ContrastResult
				fg={state.fg}
				bg={state.bg}
				ruleId={issue.ruleId}
				largeText={state.largeText}
				verdict={verdict}
				onFgChange={(value) => setState((prev) => ({ ...prev, fg: value }))}
				onBgChange={(value) => setState((prev) => ({ ...prev, bg: value }))}
				onSwap={() => setState((prev) => ({ ...prev, fg: prev.bg, bg: prev.fg }))}
				onLargeTextChange={(value) => setState((prev) => ({ ...prev, largeText: value }))}
				onRecord={(value, ratio) =>
					setVerdict(issue.id, { verdict: value, fg: state.fg, bg: state.bg, ratio })
				}
				onClear={() => clearVerdict(issue.id)}
			/>
		</div>
	);
}
