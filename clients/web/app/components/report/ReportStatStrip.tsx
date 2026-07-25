import type { UnifiedReport } from '../../lib/types/unified-report';
import {
	SEVERITY_LEVELS,
	formatScannerStatus,
	getScannerStatusTone,
	getSeverityDotClass,
	scannerLabel
} from '../../lib/report';

interface Props {
	report: UnifiedReport;
	onReviewSeverity: (severity: string) => void;
	onSelectScanner: (scannerId: string) => void;
}

export function ReportStatStrip({ report, onReviewSeverity, onSelectScanner }: Props) {
	const bySeverity = report.summary.bySeverity ?? {};

	const severityCounts = SEVERITY_LEVELS.map((level) => ({
		level,
		count: bySeverity[level] ?? 0
	})).filter((entry) => entry.count > 0);

	return (
		<div className="rstrip" aria-label="Scan summary">
			<div className="rstrip__group" aria-label="Findings by severity">
				<span className="rstrip__lab">Severity</span>
				{severityCounts.length === 0 ? (
					<span className="rstrip__clean">No findings</span>
				) : (
					severityCounts.map(({ level, count }) => (
						<button
							key={level}
							type="button"
							className={`rstrip__sev sev-badge sev-${level}`}
							onClick={() => onReviewSeverity(level)}
							title={`View ${level} findings`}
						>
							<span className={getSeverityDotClass(level)} aria-hidden="true" />
							{count.toLocaleString()} {level}
						</button>
					))
				)}
			</div>
			<div className="rstrip__group rstrip__group--scanners" aria-label="Scanner status">
				<span className="rstrip__lab">Scanners</span>
				{report.scanners.map((scanner) => {
					const issueCount = scanner.issueCount ?? report.summary.byScanner?.[scanner.id] ?? 0;
					const tone = getScannerStatusTone(scanner.status);
					const failed = scanner.status === 'failed' || !!scanner.error;
					const name = scannerLabel(scanner.id, scanner.name);
					return (
						<button
							key={scanner.id}
							type="button"
							className={`rstrip__scanner${failed ? ' rstrip__scanner--err' : ''}`}
							onClick={() => onSelectScanner(scanner.id)}
							title={
								failed
									? scanner.error || 'Scanner failed'
									: `${formatScannerStatus(scanner.status)} · filter issues by ${name}`
							}
						>
							<span className="rstrip__scanner-led" data-tone={tone} aria-hidden="true" />
							<span className="rstrip__scanner-name">{name}</span>
							<span className="rstrip__scanner-count">
								{failed ? formatScannerStatus(scanner.status) : issueCount.toLocaleString()}
							</span>
						</button>
					);
				})}
			</div>
		</div>
	);
}
