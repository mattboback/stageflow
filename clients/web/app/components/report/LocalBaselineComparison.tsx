import { ArrowRight, Check, GitCompareArrows, RotateCcw, Save } from 'lucide-react';
import { Link } from 'react-router';

import { BaselineDiffSummary } from './BaselineDiffSummary';
import { baselineCoverageProblem } from '../../lib/report/baseline-coverage';
import { shownSiteScore } from '../../lib/report';
import {
	computeProjectDiff,
	hasStableReportIdentity,
	type LocalBaseline,
	type LocalProject,
	type LocalRun
} from '../../lib/projects';
import type { UnifiedReport } from '../../lib/types/unified-report';

interface LocalBaselineComparisonProps {
	project: LocalProject;
	run: LocalRun;
	baseline: LocalBaseline | null;
	report: UnifiedReport;
	promoting: boolean;
	message: string | null;
	onPromote: () => void;
	onOpenIssue: (issueId: string) => void;
}

export function LocalBaselineComparison({
	project,
	run,
	baseline,
	report,
	promoting,
	message,
	onPromote,
	onOpenIssue
}: LocalBaselineComparisonProps) {
	const coverageProblem = baselineCoverageProblem(report);
	const isCurrentBaseline = baseline?.jobId === run.jobId;
	const compatible =
		(baseline?.configFingerprint === run.configFingerprint ||
			Boolean(baseline?.report.meta.scanContext && report.meta.scanContext)) &&
		baseline != null &&
		hasStableReportIdentity(baseline.report) === hasStableReportIdentity(report);
	const diff =
		baseline && compatible
			? computeProjectDiff(baseline.jobId, baseline.report, run.jobId, report)
			: null;

	return (
		<section className="local-baseline" aria-labelledby="local-baseline-heading">
			<div className="local-baseline__head">
				<div>
					<p className="label">Local project</p>
					<h2 id="local-baseline-heading">{project.name}</h2>
					<p>Baseline history is stored only in this browser.</p>
				</div>
				<div className="local-baseline__actions">
					<Link
						className="btn btn--ghost btn--sm"
						to={`/projects/${encodeURIComponent(project.id)}`}
					>
						Project workspace
					</Link>
					<Link
						className="btn btn--ghost btn--sm"
						to={`/playground?project=${encodeURIComponent(project.id)}`}
					>
						Run again <ArrowRight size={16} aria-hidden="true" />
					</Link>
					{/* Ghost, not primary: this sits above the score, the severity counts
					    and the review queue, so styling it as the page's call to action
					    invited promoting a baseline before seeing what was in it. The
					    confirmed fixes lead the report further down. */}
					<button
						className="btn btn--ghost btn--sm"
						type="button"
						onClick={onPromote}
						disabled={promoting || isCurrentBaseline || Boolean(coverageProblem)}
					>
						{isCurrentBaseline ? (
							<>
								<Check size={16} aria-hidden="true" /> Current baseline
							</>
						) : (
							<>
								<Save size={16} aria-hidden="true" />
								{promoting ? 'Promoting…' : baseline ? 'Replace baseline' : 'Promote as baseline'}
							</>
						)}
					</button>
				</div>
			</div>

			{coverageProblem && (
				<p className="local-baseline__message" role="status">
					{coverageProblem}
				</p>
			)}
			{message && (
				<p className="local-baseline__message" role="status">
					{message}
				</p>
			)}

			{!baseline ? (
				<div className="local-baseline__empty">
					<GitCompareArrows size={22} aria-hidden="true" />
					<div>
						<strong>No baseline yet</strong>
						<p>
							When this report looks good, promote it. The next compatible run will show regressions
							here.
						</p>
					</div>
				</div>
			) : !compatible ? (
				<div className="local-baseline__stale">
					<RotateCcw size={20} aria-hidden="true" />
					<div>
						<strong>Project configuration changed</strong>
						<p>
							The URL, scanner selection, browser options, or finding identities differ from the
							saved baseline. Review this report and promote it to start a compatible comparison.
						</p>
					</div>
				</div>
			) : diff ? (
				<BaselineDiffSummary
					diff={diff}
					shownScore={shownSiteScore(report)}
					onOpenIssue={onOpenIssue}
				/>
			) : null}
		</section>
	);
}
