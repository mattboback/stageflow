import {
	useCallback,
	useEffect,
	useLayoutEffect,
	useMemo,
	useReducer,
	useRef,
	useState,
	useSyncExternalStore
} from 'react';
import { ChevronRight, Maximize2, Minus, Plus } from 'lucide-react';

import type { IssueDetail, UnifiedReport } from '../../lib/types/unified-report';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import type { ReviewVerdict } from '../../lib/report/review-verdict';
import {
	compareSeverity,
	countIssueSeverities,
	describeSeverityCounts,
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
	newIssueIds?: ReadonlySet<string> | undefined;
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

interface Box {
	x: number;
	y: number;
	width: number;
	height: number;
}

/* What a zoom keeps in place: a point given as a fraction of the screenshot,
   and where in the viewport that point sits. Fractions rather than page
   pixels, so the native wheel listener needs nothing from the current page. */
interface ZoomAnchor {
	fx: number;
	fy: number;
	screenX: number;
	screenY: number;
}

const ZOOM_STEP = 1.25;
const ZOOM_MIN = 0.25;
const ZOOM_MAX = 4;
/* Context kept around a finding when zooming to it, in CSS pixels. */
const FINDING_MARGIN = 48;
/* Below this, a pointer press is a click on a marker; above it, a pan. Three
   pixels of tremor while clicking a 6px-stroked rect is normal. */
const DRAG_SLOP = 4;

function subscribeViewport(listener: () => void) {
	window.addEventListener('resize', listener);
	return () => window.removeEventListener('resize', listener);
}
function narrowViewport() {
	return window.innerWidth <= 880;
}
function serverViewport() {
	return false;
}

function prefersReducedMotion(): boolean {
	if (typeof window === 'undefined') return false;
	return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

function clampZoom(value: number): number {
	return Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, value));
}

function anchorAt(
	viewport: HTMLElement,
	svg: SVGSVGElement,
	screenX: number,
	screenY: number
): ZoomAnchor {
	const box = svg.getBoundingClientRect();
	const view = viewport.getBoundingClientRect();
	return {
		fx: (view.left + screenX - box.left) / box.width,
		fy: (view.top + screenY - box.top) / box.height,
		screenX,
		screenY
	};
}

function scrollToAnchor(
	viewport: HTMLElement,
	svg: SVGSVGElement,
	anchor: ZoomAnchor,
	behavior: ScrollBehavior
) {
	/* Rect deltas plus current scroll, rather than offsetLeft/offsetTop:
	   those live on HTMLElement and an SVGSVGElement does not have them. */
	const box = svg.getBoundingClientRect();
	const view = viewport.getBoundingClientRect();
	viewport.scrollTo({
		left: box.left - view.left + viewport.scrollLeft + anchor.fx * box.width - anchor.screenX,
		top: box.top - view.top + viewport.scrollTop + anchor.fy * box.height - anchor.screenY,
		behavior
	});
}

export function VisualReviewPanel({
	report,
	screenshots,
	activeScanner,
	activePage,
	newIssueIds,
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
	/* Captures are taken at deviceScaleFactor 2, so a 1280px viewport arrives
	   as a 2560px image. Zoom is kept in CSS pixels per image pixel; the label
	   speaks in page size, where 100% is the page as a visitor saw it. */
	const viewportWidth = report.meta.scanContext?.viewport?.width ?? 0;
	const captureScale =
		viewportWidth > 0 && pageWidth > viewportWidth ? Math.round(pageWidth / viewportWidth) : 1;
	const actualSize = 1 / captureScale;

	const [activeIssueId, setActiveIssueId] = useState<string | null>(null);
	const [hoveredIssueId, setHoveredIssueId] = useState<string | null>(null);
	const [rightTab, setRightTab] = useState<'pinned' | 'pagelevel'>('pinned');
	const isNarrow = useSyncExternalStore(subscribeViewport, narrowViewport, serverViewport);
	const [zoomOverride, setZoom] = useState<Zoom | null>(null);
	/* Bumped by every zoom request, so re-centering runs even when the scale
	   itself does not change: the same zoom, a different finding. */
	const [zoomRequest, bumpZoomRequest] = useReducer((count: number) => count + 1, 0);
	const anchorRef = useRef<ZoomAnchor | null>(null);
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

	/* Markers in reading order down the page rather than the paint order, which
	   is largest-first so big boxes sit behind small ones. */
	const walkOrder = useMemo(
		() => [...overlayElements].sort((a, b) => a.y - b.y || a.x - b.x),
		[overlayElements]
	);

	/* Findings on the screenshot are numbered, and their pins carry the same
	   number. Page-level findings (headers, metadata) are listed apart: there
	   is nothing on the screenshot to point at. */
	const { locatedGroups, locatedIssueIds, pageLevelGroups, pinByIssue } = useMemo(() => {
		const located = new Set(overlayElements.map((el) => el.issueId));
		const onScreenshot = issueGroups.filter((group) =>
			group.issues.some((issue) => located.has(issue.id))
		);
		const pins = new Map<string, { number: number; severity: string }>();
		onScreenshot.forEach((group, index) => {
			for (const issue of group.issues) {
				pins.set(issue.id, { number: index + 1, severity: group.severity });
			}
		});
		return {
			locatedGroups: onScreenshot,
			locatedIssueIds: located,
			pageLevelGroups: issueGroups.filter((group) => !onScreenshot.includes(group)),
			pinByIssue: pins
		};
	}, [issueGroups, overlayElements]);

	/* Fit width of a desktop capture on a phone draws the page at about 15%,
	   and 100% of the image is twice the page's real size. Small screens open
	   at real size on the first finding instead (see the layout effect below).
	   With nothing located, fit width at least shows the whole page, never its
	   empty top-left margin. */
	const zoom: Zoom = zoomOverride ?? (isNarrow && walkOrder.length > 0 ? actualSize : 'fit');
	const activeElement = walkOrder.find((el) => el.issueId === activeIssueId) ?? null;

	/*
	 * Page coordinates -> rendered pixels.
	 *
	 * The SVG carries viewBox="0 0 pageWidth pageHeight" and is laid out either
	 * at 100% of the viewport (fit) or at an explicit pageWidth * zoom, so the
	 * only honest source for the scale is what the browser actually rendered.
	 * Deriving it from `zoom` would be wrong for the whole 'fit' branch.
	 */
	const elementAnchor = useCallback(
		(element: Box, viewport: HTMLElement): ZoomAnchor => ({
			fx: (element.x + element.width / 2) / pageWidth,
			fy: (element.y + element.height / 2) / pageHeight,
			screenX: viewport.clientWidth / 2,
			screenY: viewport.clientHeight / 2
		}),
		[pageWidth, pageHeight]
	);

	const scrollToElement = useCallback(
		(element: Box, behavior: ScrollBehavior = prefersReducedMotion() ? 'auto' : 'smooth') => {
			const viewport = viewportRef.current;
			const svg = svgRef.current;
			if (!viewport || !svg || pageWidth <= 0 || pageHeight <= 0) return;
			scrollToAnchor(viewport, svg, elementAnchor(element, viewport), behavior);
		},
		[elementAnchor, pageWidth, pageHeight]
	);

	/* Every zoom keeps something in place: the active finding, the point under
	   a pinch, or else the middle of the view. The browser on its own keeps the
	   scroll offset, so each zoom slid away from what was being looked at. */
	useLayoutEffect(() => {
		const anchor = anchorRef.current;
		anchorRef.current = null;
		const viewport = viewportRef.current;
		const svg = svgRef.current;
		if (anchor && viewport && svg) scrollToAnchor(viewport, svg, anchor, 'auto');
	}, [zoomRequest]);

	/* A screenshot that opens zoomed in (small screens) starts on its first
	   finding rather than its top-left margin. Once per screenshot: after that
	   the view belongs to the reader. */
	const openedUrlRef = useRef<string | null>(null);
	useLayoutEffect(() => {
		if (!screenshotReady || openedUrlRef.current === overviewUrl) return;
		openedUrlRef.current = overviewUrl;
		const first = walkOrder[0];
		if (first && zoom !== 'fit') scrollToElement(first, 'auto');
	}, [screenshotReady, overviewUrl, walkOrder, zoom, scrollToElement]);

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
		/* The viewport only mounts once the screenshot has loaded, so this has
		   to re-run then: keyed on [] it ran while the ref was still empty and
		   pinch zoom was never attached. */
		const viewport = viewportRef.current;
		if (!viewport || !screenshotReady) return;

		const onWheel = (event: WheelEvent) => {
			if (!event.ctrlKey && !event.metaKey) return;
			event.preventDefault();
			const svg = svgRef.current;
			const view = viewport.getBoundingClientRect();
			// The point under the pinch stays under it.
			anchorRef.current = svg
				? anchorAt(viewport, svg, event.clientX - view.left, event.clientY - view.top)
				: null;
			setZoom((prev) => {
				// Leaving 'fit' or the default starts from the scale on screen.
				const current = typeof prev === 'number' ? prev : renderedScaleRef.current;
				return clampZoom(current * (event.deltaY < 0 ? ZOOM_STEP : 1 / ZOOM_STEP));
			});
			bumpZoomRequest();
		};

		viewport.addEventListener('wheel', onWheel, { passive: false });
		return () => {
			viewport.removeEventListener('wheel', onWheel);
		};
	}, [screenshotReady]);

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

	const stepToNextMarker = () => {
		if (walkOrder.length === 0) return;
		const current = walkOrder.findIndex((el) => el.issueId === activeIssueId);
		const next = walkOrder[(current + 1) % walkOrder.length];
		if (!next) return;
		setActiveIssueId(next.issueId);
		scrollToElement(next);
	};

	/* Selecting an issue has to move the canvas to it, not just mark it active.
	   Zoomed in, a marker chosen from the list is almost always outside the
	   viewport. Without this, picking an issue looked like nothing happened,
	   with no hint of which way to pan. */
	const revealIssue = (issueId: string) => {
		const element = walkOrder.find((el) => el.issueId === issueId);
		if (element) scrollToElement(element);
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

	const requestZoom = (next: Zoom | ((prev: Zoom | null) => Zoom), anchor: ZoomAnchor | null) => {
		anchorRef.current = anchor;
		setZoom(next);
		bumpZoomRequest();
	};

	/* What a button zoom keeps in place: the active finding, else the middle. */
	const focusAnchor = (): ZoomAnchor | null => {
		const viewport = viewportRef.current;
		const svg = svgRef.current;
		if (!viewport || !svg) return null;
		return activeElement
			? elementAnchor(activeElement, viewport)
			: anchorAt(viewport, svg, viewport.clientWidth / 2, viewport.clientHeight / 2);
	};

	const zoomBy = (factor: number) => {
		// Same rule as the wheel handler: leaving 'fit' has to start from the
		// scale that was on screen. Assuming 1 made the first + on a wide
		// capture jump from ~30% straight to 125%.
		requestZoom(
			(prev) => clampZoom((typeof prev === 'number' ? prev : renderedScaleRef.current) * factor),
			focusAnchor()
		);
	};

	/* Large enough to show the element with some room around it, never past
	   the page's real size, and never smaller than fit width. */
	const zoomToFinding = () => {
		const viewport = viewportRef.current;
		if (!activeElement || !viewport) return;
		const room = (size: number) => Math.max(1, size - 2 * FINDING_MARGIN);
		const boxScale = Math.min(
			room(viewport.clientWidth) / activeElement.width,
			room(viewport.clientHeight) / activeElement.height
		);
		const fitScale = viewport.clientWidth / pageWidth;
		requestZoom(
			clampZoom(Math.max(fitScale, Math.min(boxScale, actualSize))),
			elementAnchor(activeElement, viewport)
		);
	};

	const zoomPercent = zoom === 'fit' ? null : `${Math.round(zoom * captureScale * 100)}%`;

	const renderGroup = (group: IssueGroup, number: number | null) => {
		const isActive = group.issues.some((issue) => issue.id === activeIssueId);
		const reviewStatus = getReviewGroupStatus(group.issues, getReviewVerdict);
		// A group is on the screenshot when any of its issues is, so pick from
		// those: an issue without a marker gives the canvas nothing to move to.
		const located = group.issues.filter((issue) => locatedIssueIds.has(issue.id));
		const pool = located.length > 0 ? located : group.issues;
		const first =
			pool.find((issue) => needsHumanReview(issue) && !getReviewVerdict(issue.id)) ?? pool[0];
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
						revealIssue(first.id);
						onIssueSelect(first);
					}}
				>
					{number === null ? (
						<span className={getSeverityDotClass(group.severity)} aria-hidden="true" />
					) : (
						<span className={`vrev__num sev-${group.severity}`} aria-hidden="true">
							{number}
						</span>
					)}
					<span className="vrev__issue-body">
						<span className="vrev__issue-title">
							{group.title}{' '}
							{group.issues.some((issue) => newIssueIds?.has(issue.id)) && (
								<span className="issue-new">NEW</span>
							)}
						</span>
						<span className="vrev__issue-meta">
							{reviewStatus && (
								<span className={`vrev__issue-review vrev__issue-review--${reviewStatus.tone}`}>
									{reviewStatus.label}
								</span>
							)}
							{group.severity} · {group.scanner} · {group.ruleId}
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
	};

	return (
		<div className="vrev">
			<aside className="vrev__pages">
				<header className="vrev__pages-head">
					<h2>Pages</h2>
				</header>
				<ul className="vrev__pages-list">
					{report.pages.map((page) => {
						const pagePageIssues = issuesByPage[page.id] ?? [];
						const isActive = page.id === selectedPage.id;
						const counts = countIssueSeverities(pagePageIssues);
						const worst = counts[0];
						const label = page.path ?? page.url;
						const findings = `${pagePageIssues.length} ${pagePageIssues.length === 1 ? 'finding' : 'findings'}`;
						return (
							<li key={page.id}>
								<button
									type="button"
									className={`vrev__page-btn${isActive ? ' vrev__page-btn--active' : ''}`}
									aria-current={isActive ? 'true' : undefined}
									aria-label={`${label}: ${findings}${counts.length > 0 ? `, ${describeSeverityCounts(counts)}` : ''}`}
									onClick={() => onSelectPage(page.id)}
								>
									<span className="vrev__page-label" title={label}>
										{label}
									</span>
									{/* Scaled to the page with most findings, so rows compare. */}
									<span className="vrev__page-badges">
										{counts.map(({ severity, count }) => (
											<span key={severity} className="vrev__page-badge">
												<span className={getSeverityDotClass(severity)} aria-hidden="true" />
												<span className="num">{count}</span> {severity}
											</span>
										))}
									</span>
									<span className="vrev__page-meta">
										<span className="num">{findings}</span>
										{worst && (
											<>
												{' · '}
												<span className="num">{worst.count}</span> {worst.severity}
											</>
										)}
									</span>
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
								<button
									type="button"
									aria-pressed={zoom === 'fit'}
									onClick={() => requestZoom('fit', focusAnchor())}
								>
									Fit width
								</button>
								<button
									type="button"
									onClick={zoomToFinding}
									disabled={!activeElement}
									title={activeElement ? undefined : 'Select a finding on the screenshot first'}
								>
									Zoom to finding
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
						<div
							className="vrev__canvas"
							style={zoom === 'fit' ? undefined : { width: pageWidth * zoom }}
						>
							<svg
								ref={svgRef}
								className="vrev__svg"
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
									const isNew = newIssueIds?.has(el.issueId) ?? false;
									return (
										<g key={`${el.issueId}-${el.nodeIndex}`}>
											{isActive && (
												<rect
													className="vrev__halo"
													x={el.x}
													y={el.y}
													width={el.width}
													height={el.height}
													stroke={getSeverityStrokeColor(el.severity)}
													vectorEffect="non-scaling-stroke"
													aria-hidden="true"
												/>
											)}
											<rect
												x={el.x}
												y={el.y}
												width={el.width}
												height={el.height}
												fill={getSeverityFillColor(el.severity)}
												stroke={getSeverityStrokeColor(el.severity)}
												/* Screen pixels, not page pixels: at fit width a 2560px
											   capture draws at about a third, and 3px strokes thinned
											   to 1px. */
												vectorEffect="non-scaling-stroke"
												strokeWidth={isActive || isNew ? 3 : 2}
												strokeDasharray={isNew ? '6 3' : undefined}
												className={`vrev__marker${isActive ? ' vrev__marker--active' : ''}${isNew ? ' vrev__marker--new' : ''}`}
												role="button"
												tabIndex={0}
												aria-label={`${isNew ? 'NEW. ' : ''}Open finding: ${issueMap[el.issueId]?.title ?? el.issueId}`}
												style={{ cursor: 'pointer' }}
												onClick={() => handleMarkerClick(el.issueId)}
												onMouseEnter={() => setHoveredIssueId(el.issueId)}
												onMouseLeave={() => setHoveredIssueId(null)}
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
										</g>
									);
								})}
							</svg>
							{activeIssueId || hoveredIssueId ? (
								<div className="vrev__dim" aria-hidden="true" />
							) : null}
							{/* HTML, so pins keep their size at every zoom. The rects stay
						    the targets; pins only label them with the list number. */}
							<div className="vrev__pins" aria-hidden="true">
								{(() => {
									const seen = new Map<string, number>();
									const clusterSizes = new Map<string, number>();
									for (const el of walkOrder) {
										const k = `${Math.round(el.xPercent * 2)}:${Math.round(el.yPercent * 2)}`;
										clusterSizes.set(k, (clusterSizes.get(k) ?? 0) + 1);
									}
									return walkOrder.map((el) => {
										const pin = pinByIssue.get(el.issueId);
										if (!pin) return null;
										const key = `${Math.round(el.xPercent * 2)}:${Math.round(el.yPercent * 2)}`;
										const dup = seen.get(key) ?? 0;
										seen.set(key, dup + 1);
										const clusterSize = clusterSizes.get(key) ?? 1;
										if (clusterSize >= 3 && dup > 0) return null;
										if (clusterSize >= 3) {
											return (
												<span
													key={`${el.issueId}-${el.nodeIndex}`}
													className={`vrev__pin vrev__pin--cluster sev-${pin.severity}`}
													style={{ left: `${el.xPercent}%`, top: `max(1.75rem, ${el.yPercent}%)` }}
												>
													{clusterSize} findings
												</span>
											);
										}
										return (
											<span
												key={`${el.issueId}-${el.nodeIndex}`}
												className={`vrev__pin sev-${pin.severity}${el.issueId === activeIssueId ? ' vrev__pin--active' : ''}`}
												style={{
													left: `calc(${el.xPercent}% + ${dup * 26}px)`,
													top: `max(1.75rem, ${el.yPercent}%)`
												}}
											>
												{pin.number}
												{newIssueIds?.has(el.issueId) && <span className="vrev__pin-new">NEW</span>}
											</span>
										);
									});
								})()}
							</div>
						</div>
					</div>
				)}
			</section>

			<aside className="vrev__issues" aria-label="Findings on this page">
				<header className="vrev__issues-head">
					<h2>Findings on this page</h2>
					<span className="vrev__issues-count">{pageIssues.length}</span>
				</header>
				{pageIssues.length === 0 ? (
					<p className="blankslate vrev__empty-msg">
						No findings match the current scanner filter.
					</p>
				) : (
					<div className="vrev__issues-body">
						<div className="vrev__issues-toggle" role="tablist" aria-label="Finding location">
							<button
								type="button"
								role="tab"
								aria-selected={rightTab === 'pinned'}
								className={`vrev__issues-tab${rightTab === 'pinned' ? ' vrev__issues-tab--active' : ''}`}
								onClick={() => setRightTab('pinned')}
							>
								Pinned on page ({locatedGroups.reduce((s, g) => s + g.issues.length, 0)})
							</button>
							<button
								type="button"
								role="tab"
								aria-selected={rightTab === 'pagelevel'}
								className={`vrev__issues-tab${rightTab === 'pagelevel' ? ' vrev__issues-tab--active' : ''}`}
								onClick={() => setRightTab('pagelevel')}
							>
								Page-level ({pageLevelGroups.reduce((s, g) => s + g.issues.length, 0)})
							</button>
						</div>
						{[
							{
								key: 'pinned' as const,
								label: 'On the screenshot',
								groups: locatedGroups,
								numbered: true
							},
							{
								key: 'pagelevel' as const,
								label: 'Page-level',
								groups: pageLevelGroups,
								numbered: false
							}
						]
							.filter(({ key }) => key === rightTab)
							.map(
								({ label, groups, numbered }) =>
									groups.length > 0 && (
										<section key={label} className="vrev__issues-part">
											<h3 className="vrev__issues-sub">
												{label}{' '}
												<span className="num">
													{groups.reduce((sum, group) => sum + group.issues.length, 0)}
												</span>
											</h3>
											<ul className="vrev__issues-list">
												{groups.map((group, index) =>
													renderGroup(group, numbered ? index + 1 : null)
												)}
											</ul>
										</section>
									)
							)}
					</div>
				)}
			</aside>
		</div>
	);
}
