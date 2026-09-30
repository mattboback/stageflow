import { useReviewReportContext, useReviewVerdicts } from '../../lib/hooks/useReviewVerdicts';
import { ReviewNotesPanel } from './ReviewNotesPanel';
import { ScannerText } from './ScannerText';
import type { ManualCheck } from '../../lib/report';

interface Props {
	checks: ManualCheck[];
	pagesScanned: number;
}

export function ManualChecksPanel({ checks, pagesScanned }: Props) {
	const context = useReviewReportContext();
	if (checks.length === 0) return null;

	return (
		<details className="mchecks">
			<summary className="mchecks__head">
				<h3>Manual checks</h3>
				<span className="mchecks__count">{checks.length}</span>
			</summary>
			<p className="mchecks__lede">
				Lighthouse cannot automate these, so it lists them for every page it audits. They are a
				checklist for a person to work through, not findings about this site, and they are not
				counted above.
			</p>
			<ul className="mchecks__list">
				{checks.map((check) => (
					<li key={check.ruleId} className="mchecks__row">
						<p className="mchecks__title">Verify: {check.title}</p>
						<p className="mchecks__desc">
							<ScannerText text={check.description} />
						</p>
						{context &&
							check.pageIds?.map((pageId) => (
								<ManualPageReview
									key={pageId}
									jobId={context.jobId}
									subject={`manual:${check.ruleId}:${pageId}`}
									label={context.pageLabels?.[pageId] ?? pageId}
								/>
							))}
						<p className="mchecks__meta">
							lighthouse · {check.ruleId}
							{check.pageCount < pagesScanned && (
								<>
									{' '}
									· {check.pageCount} of {pagesScanned} pages
								</>
							)}
						</p>
					</li>
				))}
			</ul>
		</details>
	);
}

function ManualPageReview({
	jobId,
	subject,
	label
}: {
	jobId: string;
	subject: string;
	label: string;
}) {
	const { getVerdict, setVerdict, clearVerdict } = useReviewVerdicts(jobId);
	const review = getVerdict(subject);
	return (
		<details>
			<summary>
				{label}
				{review?.verdict ? ` · ${review.verdict}` : ' · needs review'}
			</summary>
			{review?.verdict ? (
				<button
					type="button"
					className="btn btn--quiet"
					onClick={() => {
						if (review.note) setVerdict(subject, { note: review.note });
						else clearVerdict(subject);
					}}
				>
					Clear verdict
				</button>
			) : (
				<>
					<button
						type="button"
						className="btn btn--ghost btn--sm"
						onClick={() =>
							setVerdict(subject, {
								...(review?.note ? { note: review.note } : {}),
								verdict: 'pass'
							})
						}
					>
						Mark pass
					</button>
					<button
						type="button"
						className="btn btn--ghost btn--sm"
						onClick={() =>
							setVerdict(subject, {
								...(review?.note ? { note: review.note } : {}),
								verdict: 'fail'
							})
						}
					>
						Mark fail
					</button>
				</>
			)}
			<ReviewNotesPanel jobId={jobId} issueId={subject} />
		</details>
	);
}
