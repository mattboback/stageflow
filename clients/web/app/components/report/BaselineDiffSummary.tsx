import { Check } from 'lucide-react';
import { Delta } from '../Delta';
import type { ProjectDiff } from '../../lib/projects';

export interface BaselineDiffSummaryProps {
	diff: ProjectDiff;
	heading?: string;
	onOpenIssue?: ((issueId: string) => void) | undefined;
}

export function BaselineDiffSummary({
	diff,
	heading = 'New in this deploy',
	onOpenIssue
}: BaselineDiffSummaryProps) {
	if (diff.incompatibleIdentity)
		return (
			<p className="local-baseline__message" role="status">
				Finding identities changed. Promote this report as a fresh baseline before comparing
				regressions.
			</p>
		);
	return (
		<div className="baseline-diff">
			<h2>
				{heading} ({diff.delta.newIssues}) · Fixed ({diff.delta.fixedIssues})
			</h2>
			{diff.skippedScanners && diff.skippedScanners.length > 0 && (
				<p className="local-baseline__message" role="status">
					Not compared: {diff.skippedScanners.join(', ')} have incomplete coverage in this run or
					the baseline. Missing findings are not counted as fixed.
				</p>
			)}
			<div className="local-baseline__stats" aria-label="Changes from baseline">
				<div>
					<span>Score change</span>
					<strong>
						{diff.delta.scoreDelta === undefined ? '—' : <Delta value={diff.delta.scoreDelta} />}
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
									<li key={`${issue.id}:${issue.occurrences?.[0]?.elementId ?? 'page'}`}>
										{onOpenIssue ? (
											<button
												type="button"
												onClick={() => onOpenIssue(issue.occurrences?.[0]?.elementId ?? issue.id)}
											>
												<span className="issue-new">NEW</span> {issue.title}
											</button>
										) : (
											<span>
												<span className="issue-new">NEW</span> {issue.title}
											</span>
										)}
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
									<li key={`${issue.id}:${issue.occurrences?.[0]?.elementId ?? 'page'}`}>
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
		</div>
	);
}
