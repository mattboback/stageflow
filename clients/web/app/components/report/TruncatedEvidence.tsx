import type { IssueDetail } from '../../lib/types/unified-report';

export function TruncatedEvidence({ issue }: { issue: IssueDetail }) {
	if (!issue.elementsTruncated) return null;
	const capture = issue.scannerData?.evidenceCapture;
	const meta =
		typeof capture === 'object' && capture !== null ? (capture as Record<string, unknown>) : null;
	const total = typeof meta?.totalElements === 'number' ? meta.totalElements : issue.elementCount;
	const captured =
		typeof meta?.capturedElements === 'number'
			? meta.capturedElements
			: (issue.occurrences ?? []).filter(
					(occurrence) => occurrence.html || occurrence.contextHtml || occurrence.boundingBox
				).length;
	const missing = Math.max(0, total - captured);
	return missing > 0 ? (
		<p className="evidence__truncated">
			+{missing} more elements without captured HTML or crops. All known elements are included in
			finding totals.
		</p>
	) : null;
}
