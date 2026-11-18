import { useCallback, useEffect, useState, useSyncExternalStore } from 'react';
import { MemoryRouter } from 'react-router';
import { BaselineDiffSummary } from '../components/report/BaselineDiffSummary';
import { ReportView } from '../components/report/ReportView';
import { VerificationResultBanner } from '../components/report/VerificationResultBanner';
import { useOpenIssue } from '../lib/hooks/useOpenIssue';
import { ReviewVerdictProvider } from '../lib/hooks/useReviewVerdicts';
import type { ProjectDiff } from '../lib/projects';
import {
	downloadAuditFile,
	portableScanEnvelope,
	renderPortableAuditHTML,
	type OfflineViewerAssets,
	type PortableAudit
} from '../lib/portable-audit';
import { shownSiteScore } from '../lib/report';
import type { ReviewContext } from '../lib/report/review-context';
import type { ReviewVerdictStore } from '../lib/report/review-verdict-store';

/* Rendered inside ReportView, so inside the MemoryRouter it opens findings with. */
function OfflineComparison({ diff, shownScore }: { diff: ProjectDiff; shownScore: number | null }) {
	const openIssue = useOpenIssue();
	return <BaselineDiffSummary diff={diff} shownScore={shownScore} onOpenIssue={openIssue} />;
}

export function OfflineAudit({
	audit,
	assets,
	store,
	context
}: {
	audit: PortableAudit;
	assets: OfflineViewerAssets;
	store: ReviewVerdictStore;
	context: ReviewContext;
}) {
	const subscribe = useCallback((listener: () => void) => store.subscribe(listener), [store]);
	const reviews = useSyncExternalStore(
		subscribe,
		() => store.getSnapshot(),
		() => store.getServerSnapshot()
	);
	const [savedReviews, setSavedReviews] = useState(() => JSON.stringify(audit.reviews));
	const [exportError, setExportError] = useState<string | null>(null);
	const currentReviews = JSON.stringify(reviews);
	const dirty = currentReviews !== savedReviews;
	useEffect(() => {
		if (!dirty) return;
		const warn = (event: BeforeUnloadEvent) => {
			event.preventDefault();
		};
		window.addEventListener('beforeunload', warn);
		return () => window.removeEventListener('beforeunload', warn);
	}, [dirty]);
	const updatedAudit = { ...audit, reviews };
	const baseline = audit.baseline;
	function downloadUpdated() {
		try {
			downloadAuditFile(
				`stageflow-${audit.jobId}.html`,
				renderPortableAuditHTML(updatedAudit, assets),
				'text/html'
			);
			setSavedReviews(currentReviews);
			setExportError(null);
		} catch (error) {
			setExportError(
				error instanceof Error ? error.message : 'The updated audit could not be downloaded.'
			);
		}
	}
	const artifacts = (
		<section aria-label="Portable audit artifacts">
			<h2>Saved audit data</h2>
			<p>All evidence included in this file is available offline. Live scans run in StageFlow.</p>
			<button
				className="btn btn--ghost btn--sm"
				type="button"
				onClick={() =>
					downloadAuditFile(
						'scan.json',
						JSON.stringify(portableScanEnvelope(audit.report), null, 2),
						'application/json'
					)
				}
			>
				Download scan.json
			</button>{' '}
			<button
				className="btn btn--ghost btn--sm"
				type="button"
				onClick={() =>
					downloadAuditFile('audit.json', JSON.stringify(updatedAudit), 'application/json')
				}
			>
				Download audit.json with reviews
			</button>
			{baseline && (
				<>
					{' '}
					<button
						className="btn btn--ghost btn--sm"
						type="button"
						onClick={() =>
							downloadAuditFile(
								'baseline.json',
								JSON.stringify(portableScanEnvelope(baseline), null, 2),
								'application/json'
							)
						}
					>
						Download baseline.json
					</button>
				</>
			)}
			<h3>Captured screenshots ({audit.screenshots.length})</h3>
			<ul>
				{audit.screenshots.map((screenshot) => (
					<li key={`${screenshot.page_id}:${screenshot.artifact_id}`}>
						<a
							href={screenshot.url}
							download={`${screenshot.page_id}-${screenshot.artifact_id}.webp`}
						>
							{screenshot.page_url ?? screenshot.page_id}
						</a>
					</li>
				))}
			</ul>
			{audit.missingArtifacts.length > 0 && (
				<>
					<h3>Unavailable evidence</h3>
					<p>This audit preserves the findings and reviews. These items could not be embedded:</p>
					<ul>
						{audit.missingArtifacts.map((missing, index) => (
							<li key={index}>
								{missing.pageId ? `${missing.pageId}: ` : ''}
								{missing.reason}
							</li>
						))}
					</ul>
				</>
			)}
		</section>
	);
	const toolbar = (
		<>
			<VerificationResultBanner report={audit.report} context={audit.verification ?? null} />
			<section className="offline-audit-banner" aria-label="Offline audit controls">
				<p>
					<strong>Interactive offline audit</strong> · Saved{' '}
					{new Date(audit.createdAt).toLocaleString()}
				</p>
				<p role="status">
					{dirty
						? 'Unsaved review changes. Download an updated report to keep them.'
						: 'Review changes are saved by downloading an updated report.'}
				</p>
				<button className="btn btn--ghost btn--sm" type="button" onClick={downloadUpdated}>
					Download updated report
				</button>
				{exportError && <p role="alert">{exportError}</p>}
				{audit.missingArtifacts.length > 0 && (
					<p>
						{audit.missingArtifacts.length} evidence items are unavailable. Details are listed under
						Artifacts.
					</p>
				)}
			</section>
		</>
	);
	return (
		<MemoryRouter>
			<ReviewVerdictProvider store={store} context={context}>
				<ReportView
					jobId={audit.jobId}
					report={audit.report}
					screenshots={audit.screenshots}
					status="complete"
					error={null}
					toolbar={toolbar}
					comparisonPanel={
						audit.comparison ? (
							<OfflineComparison
								diff={audit.comparison}
								shownScore={shownSiteScore(audit.report)}
							/>
						) : null
					}
					artifactsPanel={artifacts}
					{...(audit.comparison ? { comparison: audit.comparison } : {})}
				/>
			</ReviewVerdictProvider>
		</MemoryRouter>
	);
}
