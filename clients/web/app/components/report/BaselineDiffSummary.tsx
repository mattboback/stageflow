import { useId } from 'react';
import { Check, ChevronRight } from 'lucide-react';
import { Delta } from '../Delta';
import type { ProjectDiff } from '../../lib/projects';
import type { IssueDetail } from '../../lib/types/unified-report';

export interface BaselineDiffSummaryProps {
	diff: ProjectDiff;
	/**
	 * The score on the report's dial (shownSiteScore). The diff compares summary
	 * scores, as `stageflow diff` does, and a staging report's dial shows its
	 * assessment score, so the score change is left out when the two differ.
	 */
	shownScore: number | null;
	heading?: string;
	onOpenIssue?: ((issueId: string) => void) | undefined;
}

const SHOWN = 5;

function pagePath(issue: IssueDetail): string {
	try {
		return new URL(issue.pageUrl).pathname;
	} catch {
		return issue.pageUrl;
	}
}

function changeKey(issue: IssueDetail): string {
	return `${issue.id}:${issue.occurrences?.[0]?.elementId ?? 'page'}`;
}

/*
 * What changed since the baseline, in the CLI's terms: one line with the score
 * and the counts (`stageflow diff` prints "Score: 60 → 57 (-3)"), then the new
 * findings, which open in place, and the fixed ones.
 */
export function BaselineDiffSummary({
	diff,
	shownScore,
	heading = 'New in this deploy',
	onOpenIssue
}: BaselineDiffSummaryProps) {
	const headingId = useId();
	if (diff.incompatibleIdentity)
		return (
			<p className="local-baseline__message" role="status">
				Finding identities changed. Promote this report as a fresh baseline before comparing
				regressions.
			</p>
		);
	const { baseline, current, delta } = diff;
	const scored =
		delta.scoreDelta !== undefined &&
		typeof baseline.score === 'number' &&
		current.score === shownScore;
	return (
		<section className="baseline-diff" aria-labelledby={headingId}>
			<p className="label">{heading}</p>
			{/* Plain text with real separators, so it reads as one sentence. */}
			<h2 id={headingId} className="baseline-diff__title">
				{scored && (
					<>
						Score {baseline.score} → {current.score} <Delta value={delta.scoreDelta ?? 0} />
						<span className="baseline-diff__sep"> · </span>
					</>
				)}
				{delta.newIssues} new<span className="baseline-diff__sep"> · </span>
				{delta.fixedIssues} fixed<span className="baseline-diff__sep"> · </span>
				{delta.unchangedIssues} unchanged
			</h2>
			{diff.skippedScanners && diff.skippedScanners.length > 0 && (
				<p className="local-baseline__message" role="status">
					Not compared: {diff.skippedScanners.join(', ')} have incomplete coverage in this run or
					the baseline. Missing findings are not counted as fixed.
				</p>
			)}
			{diff.notComparedPages && diff.notComparedPages.length > 0 && (
				<p className="local-baseline__message" role="status">
					Not compared: some findings on {diff.notComparedPages.length} page(s) have missing or
					inconclusive checks, or different settings. Missing findings are not counted as fixed.
				</p>
			)}
			{diff.contextLimited && (
				<p className="local-baseline__message">
					This older report does not record complete execution settings.
				</p>
			)}
			{(diff.new.length > 0 || diff.fixed.length > 0) && (
				<ul className="baseline-diff__list" aria-label="Changed findings">
					{diff.new.slice(0, SHOWN).map((issue) => {
						const content = (
							<>
								<span className="issue-new">NEW</span>
								<span className="baseline-diff__name">{issue.title}</span>
								<span className="baseline-diff__where">{pagePath(issue)}</span>
							</>
						);
						return (
							<li key={changeKey(issue)}>
								{onOpenIssue ? (
									<button
										type="button"
										className="baseline-diff__row"
										onClick={() => onOpenIssue(issue.occurrences?.[0]?.elementId ?? issue.id)}
									>
										{content}
										<ChevronRight className="baseline-diff__open" size={16} aria-hidden="true" />
									</button>
								) : (
									<span className="baseline-diff__row">{content}</span>
								)}
							</li>
						);
					})}
					{diff.new.length > SHOWN && (
						<li className="baseline-diff__more">and {diff.new.length - SHOWN} more new</li>
					)}
					{diff.fixed.slice(0, SHOWN).map((issue) => (
						<li key={changeKey(issue)}>
							<span className="baseline-diff__row">
								<span className="issue-new issue-new--fixed">
									<Check size={11} strokeWidth={3} aria-hidden="true" />
									FIXED
								</span>
								<span className="baseline-diff__name">{issue.title}</span>
								<span className="baseline-diff__where">{pagePath(issue)}</span>
							</span>
						</li>
					))}
					{diff.fixed.length > SHOWN && (
						<li className="baseline-diff__more">and {diff.fixed.length - SHOWN} more fixed</li>
					)}
				</ul>
			)}
		</section>
	);
}
