import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ChevronRight, Maximize2, Minus, Plus } from 'lucide-react';

import type { IssueDetail, UnifiedReport } from '../../lib/types/unified-report';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import type { ReviewVerdict } from '../../lib/report/review-verdict';
import {
	SEVERITY_LEVELS,
	compareSeverity,
	getPageOverviewUrl,
	getReviewGroupStatus,
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
	getReviewVerdict: (issueId: string) => ReviewVerdict | null;
	onSelectPage: (pageId: string) => void;
	onIssueSelect: (issue: IssueDetail) => void;
}

interface IssueGroup {
	key: string;
	title: string;
	scanner: string;
	ruleId: string;
	severity: string;
	issues: IssueDetail[];
}

type Zoom = 'fit' | number;

const ZOOM_STEP = 1.25;
const ZOOM_MIN = 0.25;
const ZOOM_MAX = 4;
/* Below this, a pointer press is a click on a marker; above it, a pan. Three
   pixels of tremor while clicking a 6px-stroked rect is normal. */
const DRAG_SLOP = 4;

function prefersReducedMotion(): boolean {
	if (typeof window === 'undefined') return false;
	return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

export function VisualReviewPanel({
	report,
	screenshots,
	activeScanner,
	activePage,
	getReviewVerdict,
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
					issues: [issue]
				});
			}
		}
		return [...groups.values()].sort((a, b) => compareSeverity(a.severity, b.severity));
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

	const viewportRef = useRef<HTMLDivElement | null>(null);
	const svgRef = useRef<SVGSVGElement | null>(null);
	/* Set on pointerup when the press turned out to be a pan, and read by the
	   marker click handler on the click event that follows. Without it, every
	   drag that happens to start on a marker opens that finding. */
	const draggedRef = useRef(false);
	const [panning, setPanning] = useState(false);

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

	/*
	 * Page coordinates -> rendered pixels.
	 *
	 * The SVG carries viewBox="0 0 pageWidth pageHeight" and is laid out either
	 * at 100% of the viewport (fit) or at an explicit pageWidth * zoom, so the
	 * only honest source for the scale is what the browser actually rendered.
	 * Deriving it from `zoom` would be wrong for the whole 'fit' branch.
	 */
	const scrollToElement = useCallback(
		(element: { x: number; y: number; width: number; height: number }) => {
			const viewport = viewportRef.current;
			const svg = svgRef.current;
			if (!viewport || !svg || pageWidth <= 0) return;

			/* Rect deltas plus current scroll, rather than offsetLeft/offsetTop:
			   those live on HTMLElement and an SVGSVGElement does not have them. */
			const svgRect = svg.getBoundingClientRect();
			const viewRect = viewport.getBoundingClientRect();
			const scale = svgRect.width / pageWidth;
			const originX = svgRect.left - viewRect.left + viewport.scrollLeft;
			const originY = svgRect.top - viewRect.top + viewport.scrollTop;
			const centerX = originX + (element.x + element.width / 2) * scale;
			const centerY = originY + (element.y + element.height / 2) * scale;

			viewport.scrollTo({
				left: centerX - viewport.clientWidth / 2,
				top: centerY - viewport.clientHeight / 2,
				behavior: prefersReducedMotion() ? 'auto' : 'smooth'
			});
		},
		[pageWidth]
	);

	/* Leaving 'fit' has to start from the scale that was on screen, or the first
	   pinch jumps to 100% before zooming. */
	const renderedScaleRef = useRef(1);
	useEffect(() => {
		const svg = svgRef.current;
		if (!svg || pageWidth <= 0) return;
		renderedScaleRef.current = svg.getBoundingClientRect().width / pageWidth;
	});

	/*
	 * Wheel zoom, attached natively rather than through onWheel.
	 *
	 * React registers its root wheel listener as passive, so preventDefault from
	 * a synthetic handler is ignored and the browser zooms the whole page
	 * instead. Only ctrl/meta-wheel is intercepted -- that is what a trackpad
	 * pinch sends -- so ordinary two-finger scrolling still scrolls.
	 */
	useEffect(() => {
		const viewport = viewportRef.current;
		if (!viewport) return;

		const onWheel = (event: WheelEvent) => {
			if (!event.ctrlKey && !event.metaKey) return;
			event.preventDefault();
			setZoom((prev) => {
				const current = prev === 'fit' ? renderedScaleRef.current : prev;
				const next = current * (event.deltaY < 0 ? ZOOM_STEP : 1 / ZOOM_STEP);
				return Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, next));
			});
		};

		viewport.addEventListener('wheel', onWheel, { passive: false });
		return () => {
			viewport.removeEventListener('wheel', onWheel);
		};
	}, []);

	const handlePointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
		const viewport = viewportRef.current;
		// Only the primary button, and only when there is something to pan to.
		if (!viewport || event.button !== 0) return;
		if (
			viewport.scrollWidth <= viewport.clientWidth &&
			viewport.scrollHeight <= viewport.clientHeight
		)
			return;

		const startX = event.clientX;
		const startY = event.clientY;
		const startLeft = viewport.scrollLeft;
		const startTop = viewport.scrollTop;
		let moved = false;

		const onMove = (move: PointerEvent) => {
			const dx = move.clientX - startX;
			const dy = move.clientY - startY;
			if (!moved && Math.hypot(dx, dy) < DRAG_SLOP) return;
			if (!moved) {
				moved = true;
				setPanning(true);
				viewport.setPointerCapture(move.pointerId);
			}
			viewport.scrollLeft = startLeft - dx;
			viewport.scrollTop = startTop - dy;
		};

		const onUp = (up: PointerEvent) => {
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
			if (!moved) return;
			setPanning(false);
			draggedRef.current = true;
			if (viewport.hasPointerCapture(up.pointerId)) viewport.releasePointerCapture(up.pointerId);
			// Cleared after the click this drag is about to synthesize.
			window.setTimeout(() => {
				draggedRef.current = false;
			}, 0);
		};

		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
	};

	/* Markers in reading order down the page rather than the paint order, which
	   is largest-first so big boxes sit behind small ones. */
	const walkOrder = useMemo(
		() => [...overlayElements].sort((a, b) => a.y - b.y || a.x - b.x),
		[overlayElements]
	);

	const stepToNextMarker = () => {
		if (walkOrder.length === 0) return;
		const current = walkOrder.findIndex((el) => el.issueId === activeIssueId);
		const next = walkOrder[(current + 1) % walkOrder.length];
		if (!next) return;
		setActiveIssueId(next.issueId);
		scrollToElement(next);
	};

	if (!selectedPage) {
		return (
			<div className="blankslate">
				<p>No pages with screenshots available for visual review.</p>
			</div>
		);
	}

	const handleMarkerClick = (issueId: string) => {
		if (draggedRef.current) return;
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
							<div className="seg">
								<button type="button" aria-pressed={zoom === 'fit'} onClick={() => setZoom('fit')}>
									Fit width
								</button>
								<button type="button" aria-pressed={zoom === 1} onClick={() => setZoom(1)}>
									100%
								</button>
							</div>
							<button
								type="button"
								className="btn btn--icon"
								onClick={() => zoomBy(1 / ZOOM_STEP)}
								aria-label="Zoom out"
							>
								<Minus size={15} aria-hidden="true" />
							</button>
							<button
								type="button"
								className="btn btn--icon"
								onClick={() => zoomBy(ZOOM_STEP)}
								aria-label="Zoom in"
							>
								<Plus size={15} aria-hidden="true" />
							</button>
							{zoomPercent && <span className="vrev__zoom-val num">{zoomPercent}</span>}
						</div>
						{walkOrder.length > 0 && (
							<button
								type="button"
								className="vrev__stage-next"
								onClick={stepToNextMarker}
								aria-label={`Next finding on this page, ${walkOrder.length} located`}
							>
								Next finding <ChevronRight size={14} aria-hidden="true" />
							</button>
						)}
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
					<div className="blankslate">
						<p>
							No page-overview screenshot is available for this page yet. Visual review will appear
							here once captured.
						</p>
					</div>
				) : screenshotLoadFailed ? (
					<div className="blankslate">
						<p>The page-overview screenshot could not be loaded.</p>
					</div>
				) : !screenshotReady ? (
					<div className="blankslate">
						<p>Loading page-overview screenshot...</p>
					</div>
				) : (
					<div
						ref={viewportRef}
						className={`vrev__viewport${panning ? ' vrev__viewport--panning' : ''}`}
						onPointerDown={handlePointerDown}
					>
						<svg
							ref={svgRef}
							className="vrev__svg"
							style={zoom === 'fit' ? undefined : { width: pageWidth * zoom }}
							viewBox={`0 0 ${pageWidth} ${pageHeight}`}
							preserveAspectRatio="xMidYMin meet"
							role="group"
							aria-label={`Page overview for ${selectedPage.path ?? selectedPage.url}`}
						>
							<image
								aria-hidden="true"
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
										className="vrev__marker"
										role="button"
										tabIndex={0}
										aria-label={`Open finding: ${issueMap[el.issueId]?.title ?? el.issueId}`}
										style={{ cursor: 'pointer' }}
										onClick={() => handleMarkerClick(el.issueId)}
										onFocus={() => {
											setActiveIssueId(el.issueId);
											/* Tab already walks the markers; without this the
											   focused one is routinely off screen, which is the
											   same as having no keyboard support at all. */
											scrollToElement(el);
										}}
										onKeyDown={(event) => {
											if (event.key === 'Enter' || event.key === ' ') {
												event.preventDefault();
												handleMarkerClick(el.issueId);
											}
										}}
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
					<p className="blankslate vrev__empty-msg">
						No findings match the current scanner filter.
					</p>
				) : (
					<ul className="vrev__issues-list">
						{issueGroups.map((group) => {
							const isActive = group.issues.some((issue) => issue.id === activeIssueId);
							const reviewStatus = getReviewGroupStatus(group.issues, getReviewVerdict);
							const first =
								group.issues.find(
									(issue) => needsHumanReview(issue) && !getReviewVerdict(issue.id)
								) ?? group.issues[0];
							if (!first) {
								// A group only exists because an issue was added to it.
								return null;
							}
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
												{reviewStatus && (
													<span
														className={`vrev__issue-review vrev__issue-review--${reviewStatus.tone}`}
													>
														{reviewStatus.label}
													</span>
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
