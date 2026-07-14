import { useEffect, useMemo, useState } from 'react';
import { Maximize2, Minus, Plus } from 'lucide-react';

import type { IssueDetail, UnifiedReport } from '../../lib/types/unified-report';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import {
	SEVERITY_LEVELS,
	getPageOverviewUrl,
	getSeverityDotClass,
	getSeverityFillColor,
	getSeverityStrokeColor,
	needsHumanReview,
	normalizeSeverity
} from '../../lib/report';

interface Props {
	report: UnifiedReport;
	screenshots: ScreenshotArtifact[];
	activeScanner: string | null;
	activePage: string | null;
	onSelectPage: (pageId: string) => void;
	onIssueSelect: (issue: IssueDetail) => void;
}

interface IssueGroup {
	key: string;
	title: string;
	scanner: string;
	ruleId: string;
	severity: string;
	needsReview: boolean;
	issues: IssueDetail[];
}

type Zoom = 'fit' | number;

const ZOOM_STEP = 1.25;
const ZOOM_MIN = 0.25;
const ZOOM_MAX = 4;

function severityRank(severity: string | undefined): number {
	const normalized = normalizeSeverity(severity) ?? 'info';
	const index = (SEVERITY_LEVELS as readonly string[]).indexOf(normalized);
	return index === -1 ? SEVERITY_LEVELS.length : index;
}

export function VisualReviewPanel({
	report,
	screenshots,
	activeScanner,
	activePage,
	onSelectPage,
	onIssueSelect
}: Props) {
	const selectedPage = report.pages.find((p) => p.id === activePage) ?? report.pages[0] ?? null;

	const issuesByPage = useMemo(() => {
		const map: Record<string, IssueDetail[]> = {};
		for (const issue of report.issues) {
			if (activeScanner && issue.scanner !== activeScanner) continue;
			const list = map[issue.pageId] ?? [];
			list.push(issue);
			map[issue.pageId] = list;
		}
		return map;
	}, [report.issues, activeScanner]);

	const pageIssues = useMemo<IssueDetail[]>(() => {
		if (!selectedPage) return [];
		const issues = issuesByPage[selectedPage.id] ?? [];
		const seen = new Set<string>();
		return issues.filter((issue, i) => {
			const key = issue.id || String(i);
			if (seen.has(key)) return false;
			seen.add(key);
			return true;
		});
	}, [selectedPage, issuesByPage]);

	/* One row per rule: nine identical contrast findings become "× 9". */
	const issueGroups = useMemo<IssueGroup[]>(() => {
		const groups = new Map<string, IssueGroup>();
		for (const issue of pageIssues) {
			const severity = normalizeSeverity(issue.severity) ?? 'info';
			const key = `${issue.scanner}:${issue.ruleId}:${severity}`;
			const existing = groups.get(key);
			if (existing) {
				existing.issues.push(issue);
			} else {
				groups.set(key, {
					key,
					/* Manual-review items get a chip instead of a repeated
					   'Verify:' title prefix — less noise in the list. */
					title: issue.title ?? issue.ruleId,
					scanner: issue.scanner,
					ruleId: issue.ruleId,
					severity,
					needsReview: needsHumanReview(issue),
					issues: [issue]
				});
			}
		}
		return [...groups.values()].sort((a, b) => severityRank(a.severity) - severityRank(b.severity));
	}, [pageIssues]);

	const issueMap = useMemo(
		() => Object.fromEntries(pageIssues.map((issue) => [issue.id, issue])),
		[pageIssues]
	);

	const overviewUrl = selectedPage
		? getPageOverviewUrl(
				screenshots,
				selectedPage.id,
				activeScanner ? [activeScanner, 'axe'] : ['axe']
			)
		: null;

	const pageWidth = selectedPage?.pageOverview?.pageWidth ?? 0;
	const pageHeight = selectedPage?.pageOverview?.pageHeight ?? 0;
	const canRenderScreenshot = !!overviewUrl && pageWidth > 0 && pageHeight > 0;

	const [activeIssueId, setActiveIssueId] = useState<string | null>(null);
	/* Fit-width of a desktop capture on a phone renders the page at ~30% —
	   unreadable. Small screens start at 100% and pan instead. */
	const [zoom, setZoom] = useState<Zoom>(() =>
		typeof window !== 'undefined' && window.innerWidth <= 880 ? 1 : 'fit'
	);
	// Both load states are keyed by URL so switching pages needs no reset.
	const [loadedOverviewUrl, setLoadedOverviewUrl] = useState<string | null>(null);
	const [failedOverviewUrl, setFailedOverviewUrl] = useState<string | null>(null);

	useEffect(() => {
		if (!overviewUrl) {
			return;
		}

		let cancelled = false;
		const image = new Image();
		image.onload = () => {
			if (!cancelled) {
				setLoadedOverviewUrl(overviewUrl);
			}
		};
		image.onerror = () => {
			if (!cancelled) {
				setFailedOverviewUrl(overviewUrl);
			}
		};
		image.src = overviewUrl;

		return () => {
			cancelled = true;
		};
	}, [overviewUrl]);

	const screenshotReady = !!overviewUrl && loadedOverviewUrl === overviewUrl;
	const screenshotLoadFailed = !!overviewUrl && failedOverviewUrl === overviewUrl;

	const overlayElements = useMemo(() => {
		const elements = selectedPage?.pageOverview?.elements ?? [];
		return elements
			.filter((el) => Boolean(issueMap[el.issueId]))
			.sort((a, b) => b.width * b.height - a.width * a.height);
	}, [selectedPage, issueMap]);

	if (!selectedPage) {
		return (
			<div className="vrev__empty">
				<p>No pages with screenshots available for visual review.</p>
			</div>
		);
	}

	const handleMarkerClick = (issueId: string) => {
		const issue = issueMap[issueId];
		if (!issue) return;
		setActiveIssueId(issueId);
		onIssueSelect(issue);
	};

	const zoomBy = (factor: number) => {
		setZoom((prev) => {
			const current = prev === 'fit' ? 1 : prev;
			return Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, current * factor));
		});
	};

	const zoomPercent = zoom === 'fit' ? null : `${Math.round(zoom * 100)}%`;

	return (
		<div className="vrev">
			<aside className="vrev__pages">
				<header className="vrev__pages-head">
					<h3>Pages</h3>
				</header>
				<ul className="vrev__pages-list">
					{report.pages.map((page) => {
						const pagePageIssues = issuesByPage[page.id] ?? [];
						const isActive = page.id === selectedPage.id;
						const sevCounts = SEVERITY_LEVELS.map((level) => ({
							level,
							count: pagePageIssues.filter(
								(issue) => (normalizeSeverity(issue.severity) ?? 'info') === level
							).length
						})).filter((entry) => entry.count > 0);
						return (
							<li key={page.id}>
								<button
									type="button"
									className={`vrev__page-btn${isActive ? ' vrev__page-btn--active' : ''}`}
									onClick={() => onSelectPage(page.id)}
								>
									<span className="vrev__page-main">
										<span className="vrev__page-label">{page.path ?? page.url}</span>
										{sevCounts.length > 0 && (
											<span className="vrev__page-sevs" aria-hidden="true">
												{sevCounts.map(({ level, count }) => (
													<span key={level} className="vrev__page-sev">
														<span className={getSeverityDotClass(level)} />
														{count}
													</span>
												))}
											</span>
										)}
									</span>
									<span className="vrev__page-count num">{pagePageIssues.length}</span>
								</button>
							</li>
						);
					})}
				</ul>
			</aside>

			<section className="vrev__stage">
				{canRenderScreenshot && (
					<div className="vrev__stage-bar">
						<div className="vrev__zoom" role="group" aria-label="Zoom">
							<button
								type="button"
								className={`vrev__zoom-btn${zoom === 'fit' ? ' vrev__zoom-btn--on' : ''}`}
								onClick={() => setZoom('fit')}
							>
								Fit width
							</button>
							<button
								type="button"
								className={`vrev__zoom-btn${zoom === 1 ? ' vrev__zoom-btn--on' : ''}`}
								onClick={() => setZoom(1)}
							>
								100%
							</button>
							<button
								type="button"
								className="vrev__zoom-btn vrev__zoom-btn--icon"
								onClick={() => zoomBy(1 / ZOOM_STEP)}
								aria-label="Zoom out"
							>
								<Minus size={15} aria-hidden="true" />
							</button>
							<button
								type="button"
								className="vrev__zoom-btn vrev__zoom-btn--icon"
								onClick={() => zoomBy(ZOOM_STEP)}
								aria-label="Zoom in"
							>
								<Plus size={15} aria-hidden="true" />
							</button>
							{zoomPercent && <span className="vrev__zoom-val num">{zoomPercent}</span>}
						</div>
						{overviewUrl && (
							<a
								className="vrev__stage-open"
								href={overviewUrl}
								target="_blank"
								rel="noopener noreferrer"
								aria-label="Open full screenshot"
							>
								<Maximize2 size={14} aria-hidden="true" />{' '}
								<span className="vrev__stage-open-lab">Full screenshot</span>
							</a>
						)}
					</div>
				)}
				{!canRenderScreenshot ? (
					<div className="vrev__empty">
						<p>
							No page-overview screenshot is available for this page yet. Visual review will appear
							here once captured.
						</p>
					</div>
				) : screenshotLoadFailed ? (
					<div className="vrev__empty">
						<p>The page-overview screenshot could not be loaded.</p>
					</div>
				) : !screenshotReady ? (
					<div className="vrev__empty">
						<p>Loading page-overview screenshot...</p>
					</div>
				) : (
					<div className="vrev__viewport">
						<svg
							className="vrev__svg"
							style={zoom === 'fit' ? undefined : { width: pageWidth * zoom }}
							viewBox={`0 0 ${pageWidth} ${pageHeight}`}
							preserveAspectRatio="xMidYMin meet"
							role="img"
							aria-label={`Page overview for ${selectedPage.path ?? selectedPage.url}`}
						>
							<image
								href={overviewUrl ?? undefined}
								x={0}
								y={0}
								width={pageWidth}
								height={pageHeight}
								preserveAspectRatio="xMidYMin meet"
							/>
							{overlayElements.map((el) => {
								const isActive = el.issueId === activeIssueId;
								return (
									<rect
										key={`${el.issueId}-${el.nodeIndex}`}
										x={el.x}
										y={el.y}
										width={el.width}
										height={el.height}
										fill={getSeverityFillColor(el.severity)}
										stroke={getSeverityStrokeColor(el.severity)}
										strokeWidth={isActive ? 6 : 3}
										style={{ cursor: 'pointer' }}
										onClick={() => handleMarkerClick(el.issueId)}
									/>
								);
							})}
						</svg>
					</div>
				)}
			</section>

			<aside className="vrev__issues" aria-label="Findings on this page">
				<header className="vrev__issues-head">
					<h3>Findings on this page</h3>
					<span className="vrev__issues-count">{pageIssues.length}</span>
				</header>
				{pageIssues.length === 0 ? (
					<p className="vrev__empty-msg">No findings match the current scanner filter.</p>
				) : (
					<ul className="vrev__issues-list">
						{issueGroups.map((group) => {
							const isActive = group.issues.some((issue) => issue.id === activeIssueId);
							const first = group.issues[0];
							return (
								<li key={group.key}>
									<button
										type="button"
										className={`vrev__issue-btn${isActive ? ' vrev__issue-btn--active' : ''}`}
										onClick={() => {
											setActiveIssueId(first.id);
											onIssueSelect(first);
										}}
									>
										<span className={getSeverityDotClass(group.severity)} aria-hidden="true" />
										<span className="vrev__issue-body">
											<span className="vrev__issue-title">{group.title}</span>
											<span className="vrev__issue-meta">
												{group.needsReview && (
													<span className="vrev__issue-verify">needs review</span>
												)}
												{group.scanner} · {group.ruleId}
											</span>
										</span>
										{group.issues.length > 1 && (
											<span
												className="vrev__issue-times num"
												aria-label={`${group.issues.length} occurrences`}
											>
												× {group.issues.length}
											</span>
										)}
									</button>
								</li>
							);
						})}
					</ul>
				)}
			</aside>
		</div>
	);
}
