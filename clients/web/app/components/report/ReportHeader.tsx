import { AlertTriangle, FileText, Layers, ScanLine } from 'lucide-react';

import type { UnifiedReport } from '../../lib/types/unified-report';
import { Gauge } from '../Gauge';
import { Pill } from '../Pill';
import { scoreBandFor } from '../../lib/report';

interface Props {
	report: UnifiedReport;
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

export function ReportHeader({ report }: Props) {
	const score = typeof report.summary.score === 'number' ? Math.round(report.summary.score) : null;
	const band = scoreBandFor(score);
	const totals = report.summary.bySeverity;
	const critical = totals.critical ?? 0;
	const serious = totals.serious ?? 0;
	const total = report.summary.totalIssues ?? 0;
	const scannersDone = report.scanners.filter((s) => s.status === 'success').length;
	const artifactCount = report.artifacts?.length ?? 0;

	const scannedAt = formatScannedAt(report.meta.completedAt ?? report.meta.scannedAt);
	const duration = formatDuration(report.meta.durationMs);
	const baseUrl = report.meta.baseUrl;
	const host = (() => {
		if (!baseUrl) return null;
		try {
			return new URL(baseUrl).host;
		} catch {
			return baseUrl;
		}
	})();

	return (
		<header className="rhead">
			<div className="rhead__score">
				{score !== null && (
					<Gauge
						value={score}
						caption={report.summary.scoreGrade ?? band?.label ?? 'Score'}
						size={116}
						valFontSize="2rem"
					/>
				)}
				<div className="rhead__score-meta">
					<div className="rhead__title-row">
						<h1>{host ?? 'Scan report'}</h1>
						<Pill variant="done">Completed</Pill>
					</div>
					{baseUrl && (
						<a
							className="rhead__url"
							href={baseUrl}
							target="_blank"
							rel="noopener noreferrer"
						>
							{baseUrl} ↗
						</a>
					)}
					<p className="rhead__meta">
						{[scannedAt, duration].filter(Boolean).join(' · ')}
						{report.meta.jobId && (
							<>
								{scannedAt || duration ? ' · ' : ''}
								<span className="rhead__jobid mono">{report.meta.jobId.slice(0, 8)}</span>
							</>
						)}
					</p>
				</div>
			</div>
			<dl className="rhead__stats" aria-label="Scan totals">
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">
						<AlertTriangle size={15} aria-hidden="true" />
						Total issues
					</dt>
					<dd className="rhead__stat-val">{total.toLocaleString()}</dd>
					<dd className="rhead__stat-sub">
						{critical.toLocaleString()} critical · {serious.toLocaleString()} serious
					</dd>
				</div>
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">
						<FileText size={15} aria-hidden="true" />
						Pages scanned
					</dt>
					<dd className="rhead__stat-val">{report.summary.pagesScanned.toLocaleString()}</dd>
					<dd className="rhead__stat-sub">
						{report.summary.pagesWithIssues.toLocaleString()} with issues
					</dd>
				</div>
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">
						<Layers size={15} aria-hidden="true" />
						Scanners
					</dt>
					<dd className="rhead__stat-val">{report.scanners.length}</dd>
					<dd className="rhead__stat-sub">
						{scannersDone === report.scanners.length
							? 'All completed'
							: `${scannersDone} completed`}
					</dd>
				</div>
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">
						<ScanLine size={15} aria-hidden="true" />
						Artifacts
					</dt>
					<dd className="rhead__stat-val">{artifactCount}</dd>
					<dd className="rhead__stat-sub">HTML · JSON · PNG</dd>
				</div>
			</dl>
		</header>
	);
}
