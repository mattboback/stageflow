import type { UnifiedReport } from '../../lib/types/unified-report';
import { Gauge } from '../Gauge';
import { scoreBandFor } from '../../lib/report';

interface Props {
	report: UnifiedReport;
}

export function ReportHeader({ report }: Props) {
	const score = typeof report.summary.score === 'number' ? Math.round(report.summary.score) : null;
	const band = scoreBandFor(score);
	const totals = report.summary.bySeverity;
	const critical = totals.critical ?? 0;
	const serious = totals.serious ?? 0;
	const moderate = totals.moderate ?? 0;
	const minor = totals.minor ?? 0;
	const info = totals.info ?? 0;
	const total = report.summary.totalIssues ?? critical + serious + moderate + minor + info;

	return (
		<header className="rhead">
			<div className="rhead__score">
				{score !== null && (
					<Gauge
						value={score}
						caption={report.summary.scoreGrade ?? band?.label ?? 'score'}
						size={104}
						valFontSize="1.7rem"
					/>
				)}
				<div className="rhead__score-meta">
					<h1>Scan report</h1>
					<p>
						{report.summary.pagesScanned} page{report.summary.pagesScanned === 1 ? '' : 's'} ·{' '}
						{report.scanners.length} scanner{report.scanners.length === 1 ? '' : 's'}
					</p>
				</div>
			</div>
			<dl className="rhead__stats" aria-label="Issue counts">
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">Total</dt>
					<dd className="rhead__stat-val">{total}</dd>
				</div>
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">Critical</dt>
					<dd className="rhead__stat-val">{critical}</dd>
				</div>
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">Serious</dt>
					<dd className="rhead__stat-val">{serious}</dd>
				</div>
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">Moderate</dt>
					<dd className="rhead__stat-val">{moderate}</dd>
				</div>
				<div className="rhead__stat">
					<dt className="rhead__stat-lab">Minor</dt>
					<dd className="rhead__stat-val">{minor}</dd>
				</div>
			</dl>
		</header>
	);
}
