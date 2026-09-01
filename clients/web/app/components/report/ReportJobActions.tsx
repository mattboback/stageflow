import { useState, type SyntheticEvent } from 'react';
import { useNavigate } from 'react-router';

import { ConfirmDialog } from '../ConfirmDialog';
import { deleteScanJob } from '../../lib/api/client';
import { formatHostedExpiry, hostedEvidenceExpired } from '../../lib/hosted-retention';
import { IS_HOSTED_DEMO } from '../../lib/site-metadata';
import type { UnifiedReport } from '../../lib/types/unified-report';

interface ReportJobActionsProps {
	jobId: string;
	report: UnifiedReport;
	archived: boolean;
	canDelete: boolean;
	showSave: boolean;
	onSaveProject: (name: string) => Promise<void>;
}

export function ReportJobActions({
	jobId,
	report,
	archived,
	canDelete,
	showSave,
	onSaveProject
}: ReportJobActionsProps) {
	const navigate = useNavigate();
	const [copied, setCopied] = useState(false);
	const [saving, setSaving] = useState(false);
	const [saveError, setSaveError] = useState<string | null>(null);
	const [projectName, setProjectName] = useState(() => defaultProjectName(report));
	const [deleting, setDeleting] = useState(false);
	const [confirmDelete, setConfirmDelete] = useState(false);
	const [deleteError, setDeleteError] = useState<string | null>(null);
	const completedAt = report.meta.completedAt ?? report.meta.scannedAt;
	const expiryLabel = formatHostedExpiry(completedAt);
	const evidenceExpired = hostedEvidenceExpired(completedAt);

	async function copyShareLink() {
		const url = window.location.href;
		try {
			await navigator.clipboard.writeText(url);
			setCopied(true);
			window.setTimeout(() => setCopied(false), 2000);
		} catch {
			setCopied(false);
		}
	}

	async function onSave(event: SyntheticEvent) {
		event.preventDefault();
		setSaving(true);
		setSaveError(null);
		try {
			await onSaveProject(projectName.trim() || defaultProjectName(report));
		} catch (error) {
			setSaveError(error instanceof Error ? error.message : 'Could not save this report.');
		} finally {
			setSaving(false);
		}
	}

	async function onDelete() {
		setDeleting(true);
		setDeleteError(null);
		try {
			await deleteScanJob(jobId);
			void navigate('/', { replace: true });
		} catch (error) {
			setDeleteError(error instanceof Error ? error.message : 'Could not delete this scan.');
			setDeleting(false);
			setConfirmDelete(false);
		}
	}

	return (
		<section id="report-actions" className="report-job-actions" aria-label="Report actions">
			<div className="report-job-actions__meta">
				{archived ? (
					<p>
						This report is stored in this browser. Live screenshots and download links have expired.
					</p>
				) : evidenceExpired ? (
					<p>Screenshot evidence and download links have expired. Findings remain.</p>
				) : IS_HOSTED_DEMO && expiryLabel ? (
					<p>Hosted artifacts stay available until {expiryLabel}.</p>
				) : (
					<p>Share the report URL with anyone who should see this run.</p>
				)}
				<div className="report-job-actions__buttons">
					<button
						className="btn btn--ghost btn--sm"
						type="button"
						onClick={() => void copyShareLink()}
					>
						{copied ? 'Link copied' : 'Copy share link'}
					</button>
					{canDelete && !archived && (
						<button
							className="btn btn--ghost btn--sm"
							type="button"
							onClick={() => setConfirmDelete(true)}
						>
							Delete this scan
						</button>
					)}
				</div>
			</div>
			{deleteError && (
				<p className="report-job-actions__error" role="alert">
					{deleteError}
				</p>
			)}
			{showSave && (
				<form className="report-job-actions__save" onSubmit={(event) => void onSave(event)}>
					<label className="field">
						<span className="label">Save to a local project</span>
						<input
							className="input"
							value={projectName}
							onChange={(event) => setProjectName(event.target.value)}
							maxLength={80}
							autoComplete="off"
						/>
					</label>
					<button className="btn btn--ghost btn--sm" type="submit" disabled={saving}>
						{saving ? 'Saving…' : 'Save in this browser'}
					</button>
					{saveError && (
						<p className="report-job-actions__error" role="alert">
							{saveError}
						</p>
					)}
				</form>
			)}
			{confirmDelete && (
				<ConfirmDialog
					title="Delete this scan?"
					detail="Removes hosted artifacts and hides this report URL. The durable job record is not erased. A copy kept in a local project stays in this browser."
					confirmLabel={deleting ? 'Deleting…' : 'Delete scan'}
					destructive
					busy={deleting}
					onConfirm={() => void onDelete()}
					onCancel={() => setConfirmDelete(false)}
				/>
			)}
		</section>
	);
}

function defaultProjectName(report: UnifiedReport): string {
	const base = report.meta.baseUrl ?? report.pages[0]?.url;
	if (!base) return 'Saved scan';
	try {
		return new URL(base).host;
	} catch {
		return 'Saved scan';
	}
}
