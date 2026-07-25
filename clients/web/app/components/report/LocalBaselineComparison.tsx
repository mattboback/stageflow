import { ArrowRight, Check, GitCompareArrows, RotateCcw, Save } from 'lucide-react';
import { Link } from 'react-router';

import { Delta } from '../Delta';
import {
	computeProjectDiff,
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
	const isCurrentBaseline = baseline?.jobId === run.jobId;
	const compatible = baseline?.configFingerprint === run.configFingerprint;
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
					<Link className="btn btn--ghost btn--sm" to="/projects">
						All projects
					</Link>
					<Link
						className="btn btn--ghost btn--sm"
						to={`/playground?project=${encodeURIComponent(project.id)}`}
					>
						Run again <ArrowRight size={16} aria-hidden="true" />
					</Link>
					<button
						className="btn btn--primary btn--sm"
						type="button"
						onClick={onPromote}
						disabled={promoting || isCurrentBaseline}
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
							The URL, scanner selection, or browser options differ from the saved baseline. Review
							this report and promote it to start a compatible comparison.
						</p>
					</div>
				</div>
			) : diff ? (
				<>
					<div className="local-baseline__stats" aria-label="Changes from baseline">
						<div>
							<span>Score change</span>
							<strong>
								{diff.delta.scoreDelta === undefined ? (
									'—'
								) : (
									<Delta value={diff.delta.scoreDelta} />
								)}
							</strong>
						</div>
						<div className={diff.delta.newIssues > 0 ? 'local-baseline__stat--bad' : ''}>
							<span>New issues</span>
							<strong className="num">{diff.delta.newIssues}</strong>
						</div>
						<div className={diff.delta.fixedIssues > 0 ? 'local-baseline__stat--good' : ''}>
							<span>Fixed</span>
							<strong className="num">{diff.delta.fixedIssues}</strong>
						</div>
						<div>
							<span>Unchanged</span>
							<strong className="num">{diff.delta.unchangedIssues}</strong>
						</div>
					</div>

					{(diff.new.length > 0 || diff.fixed.length > 0) && (
						<div className="local-baseline__changes">
							{diff.new.length > 0 && (
								<div>
									<h3>New in this run</h3>
									<ul>
										{diff.new.slice(0, 5).map((issue) => (
											<li key={issue.id}>
												<button type="button" onClick={() => onOpenIssue(issue.id)}>
													<span className={`sev-dot sev-${issue.severity}`} aria-hidden="true" />
													{issue.title}
												</button>
											</li>
										))}
									</ul>
								</div>
							)}
							{diff.fixed.length > 0 && (
								<div>
									<h3>Fixed since baseline</h3>
									<ul>
										{diff.fixed.slice(0, 5).map((issue) => (
											<li key={issue.id}>
												<span className="local-baseline__fixed-item">
													<Check size={15} aria-hidden="true" /> {issue.title}
												</span>
											</li>
										))}
									</ul>
								</div>
							)}
						</div>
					)}
				</>
			) : null}
		</section>
	);
}
