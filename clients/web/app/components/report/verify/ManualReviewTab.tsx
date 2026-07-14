import type { IssueDetail } from '../../../lib/types/unified-report';
import { useReviewVerdicts } from '../../../lib/hooks/useReviewVerdicts';

import { ScannerText } from '../ScannerText';

interface Props {
	issue: IssueDetail;
	jobId: string;
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

export function ManualReviewTab({ issue, jobId }: Props) {
	const { getVerdict, setVerdict, clearVerdict } = useReviewVerdicts(jobId);
	const verdict = getVerdict(issue.id);

	return (
		<div className="vfy vfy--manual">
			<div className="vfy__why">
				<p className="vfy__why-head">What to review</p>
				<p className="vfy__why-body">
					<ScannerText text={issue.description} />
				</p>
				<p className="vfy__why-note">
					The scanner could not make this decision automatically. Check the page in context, then
					record the result below.
				</p>
			</div>

			{issue.helpUrl && (
				<a className="imodal__help" href={issue.helpUrl} target="_blank" rel="noopener noreferrer">
					Review the scanner guidance ↗
				</a>
			)}

			<div className="vfy__verdictbar">
				{verdict ? (
					<>
						<span className={`vfy__level-pill vfy__level-pill--${verdict.verdict}`}>
							Reviewed · {verdict.verdict}
						</span>
						<span className="vfy__verdict-meta mono">{formatVerdictTime(verdict.at)}</span>
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
						<span className="vfy__verdict-ask">Does this check pass on this page?</span>
						<button
							type="button"
							className="vfy__btn vfy__btn--pass"
							onClick={() => setVerdict(issue.id, { verdict: 'pass' })}
						>
							Mark pass
						</button>
						<button
							type="button"
							className="vfy__btn vfy__btn--fail"
							onClick={() => setVerdict(issue.id, { verdict: 'fail' })}
						>
							Mark fail
						</button>
					</>
				)}
			</div>
		</div>
	);
}
