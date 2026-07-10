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
import {
	contrastRatioFromStrings,
	describeMessageKey,
	formatRatio
} from '../../../lib/utils/contrast';
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

function formatVerdictTime(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;
	return date.toLocaleString(undefined, {
		month: 'short',
		day: 'numeric',
		hour: 'numeric',
		minute: '2-digit'
	});
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

	const ratio = contrastRatioFromStrings(state.fg, state.bg);

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

			<div className="vfy__grid">
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
					onFgChange={(value) => setState((prev) => ({ ...prev, fg: value }))}
					onBgChange={(value) => setState((prev) => ({ ...prev, bg: value }))}
					onSwap={() => setState((prev) => ({ ...prev, fg: prev.bg, bg: prev.fg }))}
					onLargeTextChange={(value) => setState((prev) => ({ ...prev, largeText: value }))}
				/>
			</div>

			{/* Decision bar sticks to the bottom of the modal scroller: the question,
			    the live result, and the actions stay adjacent while evidence scrolls. */}
			<div className="vfy__verdictbar">
				{verdict ? (
					<>
						<span
							className={`vfy__level-pill vfy__level-pill--${verdict.verdict === 'pass' ? 'pass' : 'fail'}`}
						>
							Verified · {verdict.verdict}
						</span>
						<span className="vfy__verdict-meta mono">
							{verdict.fg || '—'} on {verdict.bg || '—'}
							{verdict.ratio !== null ? ` · ${formatRatio(verdict.ratio)}:1` : ''} ·{' '}
							{formatVerdictTime(verdict.at)}
						</span>
						<button
							type="button"
							className="vfy__btn vfy__btn--ghost"
							onClick={() => clearVerdict(issue.id)}
						>
							Clear verdict
						</button>
					</>
				) : (
					<>
						<span className="vfy__verdict-ask">Does this element pass in context?</span>
						{ratio !== null && (
							<span className="vfy__verdict-ratio num">{formatRatio(ratio)}:1</span>
						)}
						<button
							type="button"
							className="vfy__btn vfy__btn--pass"
							onClick={() =>
								setVerdict(issue.id, {
									verdict: 'pass',
									fg: state.fg,
									bg: state.bg,
									ratio
								})
							}
						>
							Mark pass
						</button>
						<button
							type="button"
							className="vfy__btn vfy__btn--fail"
							onClick={() =>
								setVerdict(issue.id, {
									verdict: 'fail',
									fg: state.fg,
									bg: state.bg,
									ratio
								})
							}
						>
							Mark fail
						</button>
					</>
				)}
			</div>
		</div>
	);
}
