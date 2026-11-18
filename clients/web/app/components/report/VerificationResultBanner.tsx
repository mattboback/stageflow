import { Link } from 'react-router';
import {
	evaluateVerification,
	loadVerificationRequest,
	type VerificationRequest
} from '../../lib/verification';
import type { UnifiedReport } from '../../lib/types/unified-report';
import type { ScanStatus } from '../../lib/types/scan';
import '../../styles/verification.css';

const LABELS = {
	fixed: 'Fixed',
	'still-present': 'Still present',
	'needs-review': 'Needs review',
	unverified: 'Unverified',
	pending: 'Pending'
};

export function VerificationResultBanner({
	report,
	jobId,
	status,
	context
}: {
	report?: UnifiedReport | null;
	jobId?: string;
	status?: ScanStatus;
	context?: VerificationRequest | null;
}) {
	const currentJobId = report?.meta.jobId ?? jobId;
	const storedRequest =
		context === undefined ? (currentJobId ? loadVerificationRequest(currentJobId) : null) : context;
	const request = storedRequest && storedRequest.jobId === currentJobId ? storedRequest : null;
	if (!request && report?.meta.scanContext?.purpose !== 'verification') return null;
	const pending =
		!report &&
		status !== undefined &&
		!['complete', 'failed', 'error', 'cancelled'].includes(status);
	const result = pending
		? {
				status: 'pending' as const,
				explanation:
					'The affected page and scanner are being evaluated. A result will appear when this targeted run completes.'
			}
		: report && request
			? evaluateVerification(request, report)
			: {
					status: 'unverified' as const,
					explanation: report
						? 'The original finding context is unavailable in this browser. Review the captured evidence or reopen this scan in the browser that started verification.'
						: 'The targeted run did not produce a completed report. The original finding has not been verified; review the scan failure and retry with a working fresh build or URL.'
				};
	return (
		<section className="panel verification-result" aria-label="Targeted verification result">
			<h2>Targeted verification: {LABELS[result.status]}</h2>
			{request && <p>{request.issue.title}</p>}
			<p>{result.explanation}</p>
			<p>
				This scan covers the affected page and scanner. Run a complete audit before updating your
				project baseline.
			</p>
			{request && (
				<Link to={`/scan/${request.sourceJobId}/report`}>Return to the original report</Link>
			)}
		</section>
	);
}
