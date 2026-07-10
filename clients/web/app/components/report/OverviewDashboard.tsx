import { useMemo } from 'react';

import type { UnifiedReport } from '../../lib/types/unified-report';
import {
	formatScannerStatus,
	getScannerStatusTone,
	getSeverityDotClass,
	isManualReviewIssue,
	severityRank
} from '../../lib/report';

import { LighthouseSummary } from './LighthouseSummary';
import { SeverityBreakdown } from './SeverityBreakdown';
import { SeverityDonut } from './SeverityDonut';

interface Props {
	report: UnifiedReport;
	onSelectPage: (pageId: string) => void;
	onSelectScanner: (scannerId: string) => void;
	onSearchIssues: (query: string, scannerId?: string) => void;
	onReviewSeverity?: (severity: string) => void;
}

interface TopRule {
	key: string;
	count: number;
	title: string;
	scanner: string;
	ruleId: string;
	severity: string;
	pageIds: string[];
}

export function OverviewDashboard({
	report,
	onSelectPage,
	onSelectScanner,
	onSearchIssues,
	onReviewSeverity
}: Props) {
	const topPages = useMemo(
		() => [...report.pages].sort((a, b) => b.issueCount - a.issueCount).slice(0, 5),
		[report.pages]
	);

	const distinctIssueCount = useMemo(() => {
		const keys = new Set<string>();
		for (const issue of report.issues) keys.add(`${issue.scanner}::${issue.ruleId}`);
		return keys.size;
	}, [report.issues]);

	const repeatStats = useMemo(() => {
		const counts = new Map<string, number>();
		for (const issue of report.issues) {
			const key = `${issue.scanner}::${issue.ruleId}`;
			counts.set(key, (counts.get(key) ?? 0) + 1);
		}
		let repeats = 0;
		let patterns = 0;
		for (const count of counts.values()) {
			if (count > 1) {
				patterns += 1;
				repeats += count - 1;
			}
		}
		return { repeats, patterns };
	}, [report.issues]);

	const manualCount = useMemo(
		() => report.issues.filter(isManualReviewIssue).length,
		[report.issues]
	);

	const topRules = useMemo<TopRule[]>(() => {
		const counts: Record<
			string,
			{
				count: number;
				title: string;
				scanner: string;
				ruleId: string;
				severity: string;
				pages: Set<string>;
			}
		> = {};
		for (const issue of report.issues) {
			const key = `${issue.scanner}::${issue.ruleId}`;
			const existing = counts[key];
			if (existing) {
				existing.count += 1;
				if (issue.pageId) existing.pages.add(issue.pageId);
				if (severityRank(issue.severity) < severityRank(existing.severity)) {
					existing.severity = issue.severity;
				}
			} else {
				counts[key] = {
					count: 1,
					title: issue.title ?? issue.ruleId,
					scanner: issue.scanner,
					ruleId: issue.ruleId,
					severity: issue.severity,
					pages: new Set(issue.pageId ? [issue.pageId] : [])
				};
			}
		}
		return Object.entries(counts)
			.map(([key, meta]) => ({
				key,
				count: meta.count,
				title: meta.title,
				scanner: meta.scanner,
				ruleId: meta.ruleId,
				severity: meta.severity,
				pageIds: Array.from(meta.pages)
			}))
			.sort((a, b) => {
				const sevDiff = severityRank(a.severity) - severityRank(b.severity);
				if (sevDiff !== 0) return sevDiff;
				return b.count - a.count;
			})
			.slice(0, 5);
	}, [report.issues]);

	const urgent = useMemo(() => {
		const critical = report.summary.bySeverity?.critical ?? 0;
		if (critical > 0) return { severity: 'critical', count: critical };
		const serious = report.summary.bySeverity?.serious ?? 0;
		if (serious > 0) return { severity: 'serious', count: serious };
		return null;
	}, [report.summary.bySeverity]);

	const issueDensity =
		report.summary.pagesScanned > 0
			? report.summary.totalIssues / report.summary.pagesScanned
			: 0;
	const affectedRatio =
		report.summary.pagesScanned > 0
			? report.summary.pagesWithIssues / report.summary.pagesScanned
			: 0;
	const pagesScanned = report.summary.pagesScanned;
	const totalOccurrences = report.summary.totalIssues;
	const hasLighthouse = (report.summary.lighthouseCategories?.length ?? 0) > 0;

	const riskLabel =
		distinctIssueCount === 0
			? 'No issues found'
			: `${distinctIssueCount.toLocaleString()} issue pattern${distinctIssueCount !== 1 ? 's' : ''} found`;

	const riskSummary = (() => {
		if (distinctIssueCount === 0) {
			if ((report.errors?.length ?? 0) > 0) {
				return 'No issues detected, but some scans errored — review before treating this as clean.';
			}
			return 'No issues detected.';
		}
		const occNote =
			totalOccurrences !== distinctIssueCount
				? ` (${totalOccurrences.toLocaleString()} occurrences)`
				: '';
		return `${distinctIssueCount.toLocaleString()} issue pattern${distinctIssueCount !== 1 ? 's' : ''}${occNote} seen across ${report.summary.pagesWithIssues.toLocaleString()} of ${pagesScanned.toLocaleString()} page${pagesScanned !== 1 ? 's' : ''} scanned.`;
	})();

	const pageLabelForId = (pageId: string) => {
		const found = report.pages.find((p) => p.id === pageId);
		return found?.path ?? found?.url ?? pageId;
	};

	return (
		<div className="ovr">
			<section className="ovr__hero">
				<div className="ovr__hero-main">
					<h2 className="ovr__risk">{riskLabel}</h2>
					<p className="ovr__risk-pill">
						{report.summary.pagesWithIssues.toLocaleString()} of{' '}
						{pagesScanned.toLocaleString()} pages affected
					</p>
					<p className="ovr__summary">{riskSummary}</p>

					{distinctIssueCount > 0 && (
						<div className="ovr__workflow" data-testid="overview-workflow">
							{urgent ? (
								<button
									type="button"
									className="ovr__cta ovr__cta--urgent"
									data-severity={urgent.severity}
									onClick={() =>
										onReviewSeverity
											? onReviewSeverity(urgent.severity)
											: onSearchIssues('')
									}
								>
									Fix {urgent.count.toLocaleString()} {urgent.severity} first →
								</button>
							) : (
								<button
									type="button"
									className="ovr__cta"
									onClick={() => onSearchIssues('')}
								>
									Open issue list →
								</button>
							)}
							{repeatStats.repeats > 0 && (
								<button
									type="button"
									className="ovr__chip-btn"
									onClick={() => onSearchIssues('')}
								>
									<b>{repeatStats.repeats.toLocaleString()}</b> are repeats of{' '}
									<b>{repeatStats.patterns.toLocaleString()}</b> pattern
									{repeatStats.patterns === 1 ? '' : 's'}
								</button>
							)}
							{manualCount > 0 && (
								<button
									type="button"
									className="ovr__chip-btn"
									onClick={() => onSearchIssues('')}
								>
									<b>{manualCount.toLocaleString()}</b> need manual review
								</button>
							)}
							{topPages[0] && (
								<button
									type="button"
									className="ovr__chip-btn"
									onClick={() => onSelectPage(topPages[0]!.id)}
								>
									Inspect most-affected page
								</button>
							)}
						</div>
					)}
				</div>
				<dl className="ovr__hero-stats">
					<div>
						<dt>Pages affected</dt>
						<dd>{report.summary.pagesWithIssues}</dd>
					</div>
					<div>
						<dt>Issue density</dt>
						<dd>{issueDensity.toFixed(1)}</dd>
					</div>
					<div>
						<dt>Scanners</dt>
						<dd>{report.scanners.length}</dd>
					</div>
				</dl>
			</section>

			<div className="ovr__grid">
				<section className="ovr__card ovr__card--wide">
					<header className="ovr__card-head">
						<h3>Severity mix</h3>
					</header>
					<div className="ovr__mix">
						<SeverityDonut counts={report.summary.bySeverity} centerLabel="issues" />
						<div className="ovr__mix-break">
							<SeverityBreakdown bySeverity={report.summary.bySeverity} />
						</div>
					</div>
					<dl className="ovr__mini-stats">
						<div>
							<dt>Distinct</dt>
							<dd>{distinctIssueCount}</dd>
						</div>
						<div>
							<dt>Occurrences</dt>
							<dd>{totalOccurrences}</dd>
						</div>
						<div>
							<dt>Pages scanned</dt>
							<dd>{pagesScanned}</dd>
						</div>
						<div>
							<dt>Pages w/ issues</dt>
							<dd>{report.summary.pagesWithIssues}</dd>
						</div>
					</dl>
				</section>

				{topRules.length > 0 && (
					<section className="ovr__card ovr__card--wide">
						<header className="ovr__card-head ovr__card-head--split">
							<h3>Top fixes to start with</h3>
							<button
								type="button"
								className="ovr__link-btn"
								onClick={() => onSearchIssues('')}
							>
								Open issue list →
							</button>
						</header>
						<ol className="ovr__fixes" data-testid="top-fixes">
							{topRules.map((fix, idx) => {
								const previewPages = fix.pageIds.slice(0, 2);
								const remainingPages = Math.max(
									0,
									fix.pageIds.length - previewPages.length
								);
								return (
									<li key={fix.key}>
										<button
											type="button"
											className="ovr__fix-row"
											onClick={() => onSearchIssues(fix.ruleId, fix.scanner)}
										>
											<span className="ovr__fix-idx">#{idx + 1}</span>
											<div className="ovr__fix-body">
												<div className="ovr__fix-meta">
													<span className="ovr__fix-sev">
														<span
															className={getSeverityDotClass(fix.severity)}
															aria-hidden="true"
														/>
														{fix.severity}
													</span>
													<span className="ovr__fix-rule">
														{fix.scanner} · {fix.ruleId}
													</span>
												</div>
												<p className="ovr__fix-title">{fix.title}</p>
												<p className="ovr__fix-stats">
													{fix.count.toLocaleString()}{' '}
													{fix.count === 1 ? 'occurrence' : 'occurrences'} ·{' '}
													{fix.pageIds.length}{' '}
													{fix.pageIds.length === 1 ? 'page' : 'pages'}
												</p>
												{previewPages.length > 0 && (
													<span className="ovr__fix-pages">
														{previewPages.map((pid) => (
															<span
																key={pid}
																className="ovr__fix-page"
																title={pageLabelForId(pid)}
															>
																{pageLabelForId(pid)}
															</span>
														))}
														{remainingPages > 0 && (
															<span className="ovr__fix-more">
																+{remainingPages} more
															</span>
														)}
													</span>
												)}
											</div>
											<span className="ovr__fix-arrow" aria-hidden="true">
												→
											</span>
										</button>
									</li>
								);
							})}
						</ol>
					</section>
				)}

				<section
					className={`ovr__card ${hasLighthouse ? 'ovr__card--wide' : 'ovr__card--full'}`}
				>
					<header className="ovr__card-head">
						<h3>Scanner status</h3>
					</header>
					<div className="ovr__scanners">
						{report.scanners.map((scanner) => {
							const issueCount =
								scanner.issueCount ?? report.summary.byScanner?.[scanner.id] ?? 0;
							const hasError = scanner.status === 'failed' || !!scanner.error;
							const tone = getScannerStatusTone(scanner.status);
							return (
								<div
									key={scanner.id}
									className={`ovr__scanner${hasError ? ' ovr__scanner--err' : ''}`}
								>
									<button
										type="button"
										className="ovr__scanner-btn"
										onClick={() => onSelectScanner(scanner.id)}
									>
										<span className="ovr__scanner-main">
											<span
												className="ovr__scanner-led"
												data-tone={tone}
												aria-hidden="true"
											/>
											<span className="ovr__scanner-meta">
												<span className="ovr__scanner-name">
													{scanner.name ?? scanner.id}
												</span>
												<span className="ovr__scanner-count">
													{issueCount} issue{issueCount === 1 ? '' : 's'}
												</span>
											</span>
										</span>
										<span className="ovr__scanner-status" data-tone={tone}>
											{formatScannerStatus(scanner.status)}
										</span>
									</button>
									{hasError && (
										<div className="ovr__scanner-err-msg">
											<p className="ovr__scanner-err-lab">Scanner failure</p>
											<p>{scanner.error || 'Terminated without producing results.'}</p>
										</div>
									)}
								</div>
							);
						})}
					</div>
				</section>

				{hasLighthouse && (
					<section className="ovr__card ovr__card--wide">
						<header className="ovr__card-head">
							<h3>Lighthouse averages</h3>
						</header>
						<div className="ovr__card-body">
							<LighthouseSummary
								categories={report.summary.lighthouseCategories ?? []}
							/>
						</div>
					</section>
				)}

				<section className="ovr__card">
					<header className="ovr__card-head">
						<h3>Most affected pages</h3>
					</header>
					{topPages.length === 0 ? (
						<div className="ovr__empty">No pages scanned.</div>
					) : (
						topPages.map((page) => (
							<button
								key={page.id}
								type="button"
								className="ovr__page-row"
								onClick={() => onSelectPage(page.id)}
							>
								<span className="ovr__page-label">{page.path ?? page.url}</span>
								<span className="ovr__page-count">
									{page.issueCount} issue{page.issueCount === 1 ? '' : 's'}
								</span>
							</button>
						))
					)}
				</section>

				<section className="ovr__card">
					<header className="ovr__card-head">
						<h3>Patterns to fix</h3>
					</header>
					<div className="ovr__patterns">
						{topRules.map((rule) => (
							<button
								key={rule.key}
								type="button"
								className="ovr__pattern-row"
								onClick={() => onSearchIssues(rule.ruleId, rule.scanner)}
							>
								<span className="ovr__pattern-title">{rule.title}</span>
								<span className="ovr__pattern-meta">
									{rule.scanner} · {rule.count} occurrence
									{rule.count === 1 ? '' : 's'}
								</span>
							</button>
						))}
					</div>
				</section>

				<section className="ovr__card ovr__card--wide">
					<header className="ovr__card-head">
						<h3>Coverage</h3>
					</header>
					<div className="ovr__coverage">
						<div>
							<div className="ovr__coverage-row">
								<span>Pages with issues</span>
								<span>{Math.round(affectedRatio * 100)}%</span>
							</div>
							<div className="ovr__coverage-bar">
								<span
									className="ovr__coverage-fill"
									data-tone={
										affectedRatio > 0.75
											? 'high'
											: affectedRatio > 0.5
												? 'mid'
												: 'low'
									}
									style={{ transform: `scaleX(${affectedRatio})` }}
								/>
							</div>
						</div>
						<div>
							<div className="ovr__coverage-row">
								<span>Distinct issues</span>
								<span>{distinctIssueCount.toLocaleString()}</span>
							</div>
							<p className="ovr__coverage-note">
								{totalOccurrences.toLocaleString()} total occurrence
								{totalOccurrences !== 1 ? 's' : ''} across {report.scanners.length}{' '}
								scanner{report.scanners.length !== 1 ? 's' : ''}.
							</p>
						</div>
					</div>
				</section>
			</div>
		</div>
	);
}
