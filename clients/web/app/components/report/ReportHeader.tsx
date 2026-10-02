import { AlertTriangle, FileText } from 'lucide-react';

import type { UnifiedReport } from '../../lib/types/unified-report';
import { Gauge } from '../Gauge';
import { Pill } from '../Pill';
import { incompleteScanners, scannerLabel, scoreBandFor } from '../../lib/report';

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
	return date.toLocaleString('en-US', {
		timeZone: 'UTC',
		month: 'short',
		day: 'numeric',
		year: 'numeric',
		hour: 'numeric',
		minute: '2-digit',
		timeZoneName: 'short'
	});
}

export function ReportHeader({ report }: Props) {
	const selectedScore = report.assessment?.score ?? report.summary.score;
	const score =
		report.meta.scanContext?.purpose !== 'verification' && typeof selectedScore === 'number'
			? Math.round(selectedScore)
			: null;
	const band = scoreBandFor(score);
	const totals = report.summary.bySeverity;
	const critical = totals.critical ?? 0;
	const serious = totals.serious ?? 0;
	const total = report.summary.totalIssues ?? 0;
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

	const grade = report.assessment?.scoreGrade ?? report.summary.scoreGrade ?? band?.label ?? null;
	const missing = incompleteScanners(report.scanners).map((id) =>
		scannerLabel(id, report.scanners.find((scanner) => scanner.id === id)?.name)
	);

	return (
		<header className="rhead">
			<div className="rhead__score">
				{score !== null && missing.length === 0 && (
					<Gauge
						value={score}
						// The ring fits a short caption; the explanation below says the
						// score is automated.
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
						{report.meta.profile && (
							<Pill variant="done">
								{report.meta.profile === 'staging' ? 'Staging profile' : 'Production profile'}
							</Pill>
						)}
						{report.meta.scanContext?.purpose === 'verification' && (
							<Pill variant="error">Targeted verification</Pill>
						)}
						{missing.length > 0 ? (
							<Pill variant="error">Partial</Pill>
						) : (
							<Pill variant="done">Completed</Pill>
						)}
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
					{score !== null && missing.length === 0 && (
						<p className="rhead__score-explanation">
							Automated score weights actionable findings by severity; expected staging observations
							and human checks need separate review.
						</p>
					)}
					{report.assessment && (
						<p>
							{report.assessment.actionable} actionable · {report.assessment.expected} expected ·{' '}
							{report.assessment.needsReview} need human review
						</p>
					)}
					{missing.length > 0 && (
						<p className="rhead__partial" role="status">
							{missing.join(', ')} did not finish, so {missing.length === 1 ? 'its' : 'their'}{' '}
							findings are missing and no site score is given.
						</p>
					)}
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
							? missing.length > 0
								? 'coverage incomplete'
								: 'nothing to fix'
							: `${critical.toLocaleString()} critical · ${serious.toLocaleString()} serious`}
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
