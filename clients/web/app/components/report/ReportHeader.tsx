import { AlertTriangle, Eye, FileText } from 'lucide-react';

import type { UnifiedReport } from '../../lib/types/unified-report';
import { Gauge } from '../Gauge';
import { Pill } from '../Pill';
import { scoreBandFor, type ReviewProgress } from '../../lib/report';

interface Props {
	report: UnifiedReport;
	reviewProgress: ReviewProgress;
}

function formatDuration(ms: number | undefined): string | null {
	if (typeof ms !== 'number' || ms <= 0) return null;
	const totalSeconds = Math.round(ms / 1000);
	const m = Math.floor(totalSeconds / 60);
	const s = totalSeconds % 60;
	return m > 0 ? `${m}m ${s}s` : `${s}s`;
}

function formatScannedAt(iso: string | undefined): string | null {
	if (!iso) return null;
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return null;
	return date.toLocaleString(undefined, {
		month: 'short',
		day: 'numeric',
		year: 'numeric',
		hour: 'numeric',
		minute: '2-digit'
	});
}

export function ReportHeader({ report, reviewProgress }: Props) {
	const score = typeof report.summary.score === 'number' ? Math.round(report.summary.score) : null;
	const band = scoreBandFor(score);
	const totals = report.summary.bySeverity;
	const critical = totals.critical ?? 0;
	const serious = totals.serious ?? 0;
	const total = report.summary.totalIssues ?? 0;
	const pendingCopy =
		reviewProgress.pending === 1
			? '1 finding needs a human check'
			: `${reviewProgress.pending} findings need a human check`;
	const reviewedCopy =
		reviewProgress.reviewed === 1
			? 'all 1 finding reviewed'
			: `all ${reviewProgress.reviewed} findings reviewed`;

	const scannedAt = formatScannedAt(report.meta.completedAt ?? report.meta.scannedAt);
	const duration = formatDuration(report.meta.durationMs);
	const baseUrl = (() => {
		if (report.meta.baseUrl) return report.meta.baseUrl;
		const firstPage = report.pages[0]?.url;
		if (!firstPage) return undefined;
		try {
			return new URL(firstPage).origin;
		} catch {
			return firstPage;
		}
	})();
	const host = (() => {
		if (!baseUrl) return null;
		try {
			return new URL(baseUrl).host;
		} catch {
			return baseUrl;
		}
	})();

	const grade = report.summary.scoreGrade ?? band?.label ?? null;

	return (
		<header className="rhead">
			<div className="rhead__score">
				{score !== null && (
					<Gauge
						value={score}
						caption={grade ? `${grade} · Site score` : 'Site score'}
						size={92}
						valFontSize="1.7rem"
					/>
				)}
				<div className="rhead__score-meta">
					<h1>{host ?? 'Scan report'}</h1>
					{baseUrl && (
						<a className="rhead__url" href={baseUrl} target="_blank" rel="noopener noreferrer">
							{baseUrl} ↗
						</a>
					)}
					<div className="rhead__meta">
						<Pill variant="done">Completed</Pill>
						{(scannedAt || duration || report.meta.jobId) && (
							<details className="rhead__details">
								<summary>Run details</summary>
								<span className="rhead__details-body">
									{[scannedAt, duration].filter(Boolean).join(' · ')}
									{report.meta.jobId && (
										<>
											{scannedAt || duration ? ' · ' : ''}
											<span className="rhead__jobid mono">job {report.meta.jobId}</span>
										</>
									)}
								</span>
							</details>
						)}
					</div>
				</div>
			</div>
			<dl className="rhead__stats" aria-label="Scan totals">
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">
						<AlertTriangle size={16} aria-hidden="true" />
						Findings
					</dt>
					<dd className="rhead__stat-val">{total.toLocaleString()}</dd>
					<dd className="rhead__stat-sub">
						{total === 0
							? 'nothing to fix'
							: `${critical.toLocaleString()} critical · ${serious.toLocaleString()} serious`}
					</dd>
				</div>
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">
						<Eye size={16} aria-hidden="true" />
						Human review
					</dt>
					<dd className="rhead__stat-val">{reviewProgress.total.toLocaleString()}</dd>
					<dd className="rhead__stat-sub">
						{reviewProgress.pending > 0
							? reviewProgress.reviewed > 0
								? `${reviewProgress.reviewed} reviewed · ${pendingCopy}`
								: pendingCopy
							: reviewProgress.total > 0
								? reviewedCopy
								: 'nothing to verify'}
					</dd>
				</div>
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">
						<FileText size={16} aria-hidden="true" />
						Pages scanned
					</dt>
					<dd className="rhead__stat-val">{report.summary.pagesScanned.toLocaleString()}</dd>
					<dd className="rhead__stat-sub">
						{report.summary.pagesWithIssues.toLocaleString()} with findings
					</dd>
				</div>
			</dl>
		</header>
	);
}
