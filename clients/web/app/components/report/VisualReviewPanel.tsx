import { useEffect, useMemo, useState } from 'react';

import type {
	IssueDetail,
	UnifiedReport
} from '../../lib/types/unified-report';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import {
	getPageOverviewUrl,
	getSeverityDotClass,
	getSeverityFillColor,
	getSeverityStrokeColor,
	rewriteIssueTitle
} from '../../lib/report';

interface Props {
	report: UnifiedReport;
	screenshots: ScreenshotArtifact[];
	activeScanner: string | null;
	activePage: string | null;
	onSelectPage: (pageId: string) => void;
	onIssueSelect: (issue: IssueDetail) => void;
}

export function VisualReviewPanel({
	report,
	screenshots,
	activeScanner,
	activePage,
	onSelectPage,
	onIssueSelect
}: Props) {
	const selectedPage =
		report.pages.find((p) => p.id === activePage) ?? report.pages[0] ?? null;

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

	return (
		<div className="vrev">
			<aside className="vrev__pages">
				<header className="vrev__pages-head">
					<h3>Pages</h3>
				</header>
				<ul className="vrev__pages-list">
					{report.pages.map((page) => {
						const count = (issuesByPage[page.id] ?? []).length;
						const isActive = page.id === selectedPage.id;
						return (
							<li key={page.id}>
								<button
									type="button"
									className={`vrev__page-btn${isActive ? ' vrev__page-btn--active' : ''}`}
									onClick={() => onSelectPage(page.id)}
								>
									<span className="vrev__page-label">{page.path ?? page.url}</span>
									<span className="vrev__page-count">{count}</span>
								</button>
							</li>
						);
					})}
				</ul>
			</aside>

			<section className="vrev__stage">
				{!canRenderScreenshot ? (
					<div className="vrev__empty">
						<p>
							No page-overview screenshot is available for this page yet. Visual review
							will appear here once captured.
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

			<aside className="vrev__issues" aria-label="Page issues">
				<header className="vrev__issues-head">
					<h3>Issues on this page</h3>
					<span className="vrev__issues-count">{pageIssues.length}</span>
				</header>
				{pageIssues.length === 0 ? (
					<p className="vrev__empty-msg">No issues match the current scanner filter.</p>
				) : (
					<ul className="vrev__issues-list">
						{pageIssues.map((issue) => {
							const isActive = issue.id === activeIssueId;
							return (
								<li key={issue.id}>
									<button
										type="button"
										className={`vrev__issue-btn${isActive ? ' vrev__issue-btn--active' : ''}`}
										onClick={() => {
											setActiveIssueId(issue.id);
											onIssueSelect(issue);
										}}
									>
										<span
											className={getSeverityDotClass(issue.severity)}
											aria-hidden="true"
										/>
										<span className="vrev__issue-body">
											<span className="vrev__issue-title">
												{rewriteIssueTitle(issue)}
											</span>
											<span className="vrev__issue-meta">
												{issue.scanner} · {issue.ruleId}
											</span>
										</span>
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
