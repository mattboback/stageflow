import { useMemo } from 'react';

import type { UnifiedReport } from '../../lib/types/unified-report';
import {
	SEVERITY_LEVELS,
	formatScannerStatus,
	getScannerStatusTone,
	getSeverityDotClass,
	isManualReviewIssue,
	scannerLabel
} from '../../lib/report';
import { useContrastVerdicts } from '../../lib/hooks/useContrastVerdicts';

interface Props {
	report: UnifiedReport;
	jobId: string;
	onReviewSeverity: (severity: string) => void;
	onSelectScanner: (scannerId: string) => void;
	onReviewManual: () => void;
}

export function ReportStatStrip({
	report,
	jobId,
	onReviewSeverity,
	onSelectScanner,
	onReviewManual
}: Props) {
	const bySeverity = report.summary.bySeverity ?? {};
	const { getVerdict } = useContrastVerdicts(jobId);
	const manualIssues = useMemo(
		() => report.issues.filter(isManualReviewIssue),
		[report.issues]
	);
	const manualCount = manualIssues.length;
	const reviewedCount = manualIssues.filter((issue) => getVerdict(issue.id)).length;

	const severityCounts = SEVERITY_LEVELS.map((level) => ({
		level,
		count: bySeverity[level] ?? 0
	})).filter((entry) => entry.count > 0);

	return (
		<div className="rstrip" aria-label="Scan summary">
			<div className="rstrip__group" aria-label="Issues by severity">
				<span className="rstrip__lab">Severity</span>
				{severityCounts.length === 0 ? (
					<span className="rstrip__clean">No issues found</span>
				) : (
					severityCounts.map(({ level, count }) => (
						<button
							key={level}
							type="button"
							className={`rstrip__sev sev-badge sev-${level}`}
							onClick={() => onReviewSeverity(level)}
							title={`Review ${level} issues`}
						>
							<span className={getSeverityDotClass(level)} aria-hidden="true" />
							{count.toLocaleString()} {level}
						</button>
					))
				)}
			</div>
			{manualCount > 0 && (
				<div className="rstrip__group" aria-label="Review queue">
					<span className="rstrip__lab">Review queue</span>
					<button
						type="button"
						className="rstrip__manual"
						onClick={onReviewManual}
						title="Review issues needing manual verification"
					>
						{reviewedCount > 0
							? `${reviewedCount.toLocaleString()} of ${manualCount.toLocaleString()} reviewed`
							: `${manualCount.toLocaleString()} need manual review`}{' '}
						<span className="ar" aria-hidden="true">
							→
						</span>
					</button>
				</div>
			)}
			<div className="rstrip__group rstrip__group--scanners" aria-label="Scanner status">
				<span className="rstrip__lab">Scanners</span>
				{report.scanners.map((scanner) => {
					const issueCount =
						scanner.issueCount ?? report.summary.byScanner?.[scanner.id] ?? 0;
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
