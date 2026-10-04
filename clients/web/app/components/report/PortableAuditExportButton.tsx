import { useReviewSnapshot } from '../../lib/hooks/useReviewVerdicts';
import { useState } from 'react';
import {
	downloadAuditFile,
	loadOfflineViewerAssets,
	preparePortableAudit,
	renderPortableAuditHTML,
	type PortableAuditInput
} from '../../lib/portable-audit';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import { loadVerificationRequest } from '../../lib/verification';

export interface PortableAuditExportButtonProps extends PortableAuditInput {
	refreshScreenshots?: () => Promise<ScreenshotArtifact[]>;
}

export function PortableAuditExportButton({
	refreshScreenshots,
	...input
}: PortableAuditExportButtonProps) {
	const reviews = useReviewSnapshot();
	const [busy, setBusy] = useState(false);
	const [message, setMessage] = useState<string | null>(null);
	async function exportAudit() {
		setBusy(true);
		setMessage(null);
		try {
			let screenshots = input.screenshots;
			let refreshFailed = false;
			if (refreshScreenshots) {
				try {
					screenshots = await refreshScreenshots();
				} catch {
					refreshFailed = true;
				}
			}
			const assets = await loadOfflineViewerAssets();
			const verification = input.verification ?? loadVerificationRequest(input.jobId);
			const audit = await preparePortableAudit({
				...input,
				reviews: input.reviews ?? reviews,
				screenshots,
				...(verification ? { verification } : {})
			});
			if (refreshFailed)
				audit.missingArtifacts.push({
					pageId: '',
					artifactId: '',
					reason: 'Artifact links could not be refreshed; available evidence was exported.'
				});
			downloadAuditFile(
				`stageflow-${input.jobId}.html`,
				renderPortableAuditHTML(audit, assets),
				'text/html'
			);
			setMessage(
				audit.missingArtifacts.length > 0
					? `Audit downloaded with ${audit.missingArtifacts.length} unavailable evidence items. See the artifact list in the downloaded report.`
					: 'Interactive audit downloaded. Open the HTML file to review it offline.'
			);
		} catch (error) {
			setMessage(error instanceof Error ? error.message : 'The audit could not be downloaded.');
		} finally {
			setBusy(false);
		}
	}
	return (
		<div className="portable-audit-export">
			<button
				className="btn btn--ghost btn--sm"
				type="button"
				disabled={busy}
				onClick={() => {
					void exportAudit();
				}}
			>
				{busy ? 'Preparing offline audit…' : 'Download interactive audit'}
			</button>
			{message && <p role="status">{message}</p>}
		</div>
	);
}
