import type { ProjectDiff } from '../../lib/projects';
import type { UnifiedReport } from '../../lib/types/unified-report';
import { Delta } from '../Delta';
import { Gauge } from '../Gauge';
import { Pill } from '../Pill';
import {
	formatScannerStatus,
	getScannerStatusTone,
	incompleteScanners,
	scannerLabel,
	scoreBandFor,
	severityCounts,
	shownSiteScore,
	type ScoreModel
} from '../../lib/report';
import { ScoreExplainer } from './ScoreExplainer';
import { SeverityBar, SeverityLegend } from './SeverityBar';

/** A break opportunity before each dot, so a long host wraps between labels, not mid-word. */
function breakableHost(host: string) {
	return host
		.split('.')
		.flatMap((label, index) => (index === 0 ? [label] : [<wbr key={index} />, `.${label}`]));
}

interface Props {
	report: UnifiedReport;
	/** explainScore of the report as stored; with it the score says what lowers it. */
	scoreModel?: ScoreModel | null | undefined;
	/** A baseline comparison: the dial marks the baseline score and the change. */
	comparison?: ProjectDiff | null | undefined;
	/** Severity legend entries become filters for the findings list. */
	onReviewSeverity?: ((severity: string) => void) | undefined;
	/** Scanner chips become filters for the findings list. */
	onSelectScanner?: ((scannerId: string) => void) | undefined;
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

export function ReportHeader({
	report,
	scoreModel,
	comparison,
	onReviewSeverity,
	onSelectScanner
}: Props) {
	const shownScore = shownSiteScore(report);
	const band = scoreBandFor(shownScore);
	const counts = severityCounts(report.summary.bySeverity);
	const total = report.summary.totalIssues ?? 0;
	const pagesScanned = report.summary.pagesScanned;
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
	// The diff compares summary scores. Mark the baseline only when the
	// current side is the number on the dial, so the two can't disagree.
	const baselineScore =
		shownScore !== null && comparison?.current.score === shownScore
			? comparison.baseline.score
			: undefined;
	const assessment = report.assessment;

	return (
		<header className="rhead">
			<div className="rhead__score">
				{shownScore !== null && (
					<div className="rhead__dial">
						<Gauge
							value={shownScore}
							// The ring fits a short caption; the disclosure beside it explains
							// what the score counts.
							caption={grade ? `${grade} · Site score` : 'Site score'}
							size={132}
							valFontSize="1.7rem"
							baseline={baselineScore}
						/>
						{baselineScore !== undefined && (
							// The dial's accessible name already carries the change.
							<p className="rhead__delta" aria-hidden="true">
								<Delta value={shownScore - baselineScore} /> <span>vs {baselineScore}</span>
							</p>
						)}
					</div>
				)}
				<div className="rhead__score-meta">
					<h1>{host ? breakableHost(host) : 'Scan report'}</h1>
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
					{assessment && assessment.expected + assessment.needsReview > 0 && (
						<p className="rhead__assessment">
							{assessment.actionable} actionable · {assessment.expected} expected ·{' '}
							{assessment.needsReview} need human review
						</p>
					)}
					{shownScore !== null && (
						<ScoreExplainer
							model={scoreModel}
							scanners={report.scanners}
							excludesExpected={report.assessment?.score !== undefined}
						/>
					)}
					{missing.length > 0 && (
						<p className="rhead__partial" role="status">
							{missing.join(', ')} did not finish, so {missing.length === 1 ? 'its' : 'their'}{' '}
							findings are missing and no site score is given.
						</p>
					)}
				</div>
			</div>
			<div className="rhead__totals" role="group" aria-label="Scan totals">
				<p className="rhead__total">
					<span className="rhead__total-val num">{total.toLocaleString()}</span>{' '}
					{total === 1 ? 'finding' : 'findings'}
				</p>
				<p className="rhead__total-sub">
					{pagesScanned.toLocaleString()} {pagesScanned === 1 ? 'page' : 'pages'} scanned ·{' '}
					{total > 0
						? `${report.summary.pagesWithIssues.toLocaleString()} with findings`
						: missing.length > 0
							? 'coverage incomplete'
							: 'nothing to fix'}
				</p>
				{counts.length > 0 && (
					<>
						<SeverityBar counts={counts} size="lg" />
						<SeverityLegend
							counts={counts}
							onSelect={onReviewSeverity}
							selectTitle={(severity) => `View ${severity} findings`}
						/>
					</>
				)}
			</div>
			{report.scanners.length > 0 && (
				<div className="rhead__scanners" role="group" aria-label="Scanners">
					<span className="rhead__scanners-lab">Scanners</span>
					{report.scanners.map((scanner) => {
						const issueCount = scanner.issueCount ?? report.summary.byScanner?.[scanner.id] ?? 0;
						const failed = scanner.status === 'failed' || !!scanner.error;
						const name = scannerLabel(scanner.id, scanner.name);
						const content = (
							<>
								<span
									className="rhead__scanner-led"
									data-tone={getScannerStatusTone(scanner.status)}
									aria-hidden="true"
								/>
								<span className="rhead__scanner-name">{name}</span>
								<span className="rhead__scanner-count num">
									{failed ? formatScannerStatus(scanner.status) : issueCount.toLocaleString()}
								</span>
							</>
						);
						const className = `rhead__scanner${failed ? ' rhead__scanner--err' : ''}`;
						return onSelectScanner ? (
							<button
								key={scanner.id}
								type="button"
								className={className}
								onClick={() => onSelectScanner(scanner.id)}
								title={
									failed
										? scanner.error || 'Scanner failed'
										: `${formatScannerStatus(scanner.status)} · filter findings by ${name}`
								}
							>
								{content}
							</button>
						) : (
							<span key={scanner.id} className={className}>
								{content}
							</span>
						);
					})}
				</div>
			)}
		</header>
	);
}
