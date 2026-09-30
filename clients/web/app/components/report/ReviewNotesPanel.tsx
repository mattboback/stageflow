import { useState } from 'react';
import { useReviewVerdicts } from '../../lib/hooks/useReviewVerdicts';
import { MAX_REVIEW_NOTE_LENGTH, formatVerdictTime } from '../../lib/report/review-verdict';

export function ReviewNotesPanel({ jobId, issueId }: { jobId: string; issueId: string }) {
	const { getVerdict, getPriorReview, confirmPriorReview, setNote } = useReviewVerdicts(jobId);
	const review = getVerdict(issueId);
	const prior = getPriorReview(issueId);
	const [draft, updateNote] = useState<string | null>(null);
	const note = draft ?? review?.note ?? '';
	const [saved, setSaved] = useState(false);
	return (
		<section className="imodal__pane" aria-label="Review notes">
			{!review?.verdict && prior && (
				<div role="status">
					<p>
						Previously reviewed {prior.review.verdict ? `· ${prior.review.verdict}` : ''} ·{' '}
						{formatVerdictTime(prior.review.at)}
					</p>
					{prior.review.note && <p>{prior.review.note}</p>}
					{prior.stale ? (
						<p>Evidence or scan settings changed, or evidence is missing. Review this run again.</p>
					) : (
						<>
							<p>Confirm this decision for the current run.</p>
							<button
								type="button"
								className="btn btn--ghost btn--sm"
								onClick={() => {
									confirmPriorReview(issueId);
									updateNote(prior.review.note ?? '');
								}}
							>
								Confirm previous review
							</button>
						</>
					)}
				</div>
			)}
			<label htmlFor={`review-note-${issueId}`}>Review note (optional)</label>
			<textarea
				id={`review-note-${issueId}`}
				value={note}
				maxLength={MAX_REVIEW_NOTE_LENGTH}
				onChange={(event) => {
					updateNote(event.target.value);
					setSaved(false);
				}}
				rows={3}
			/>
			<button
				type="button"
				className="btn btn--ghost btn--sm"
				onClick={() => {
					setNote(issueId, note);
					setSaved(true);
				}}
			>
				Save note
			</button>
			{saved && <span role="status">Note saved.</span>}
			<p>Human reviews do not change the automated score or regression checks.</p>
		</section>
	);
}
