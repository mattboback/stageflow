import {
	useCallback,
	useEffect,
	useLayoutEffect,
	useReducer,
	useRef,
	useState,
	useSyncExternalStore
} from 'react';

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

interface ViewportOptions {
	pageWidth: number;
	pageHeight: number;
	actualSize: number;
	activeElement: Box | null;
	firstElement: Box | undefined;
	overviewUrl: string | null;
	screenshotReady: boolean;
}

export function useScreenshotViewport({
	pageWidth,
	pageHeight,
	actualSize,
	activeElement,
	firstElement,
	overviewUrl,
	screenshotReady
}: ViewportOptions) {
	const isNarrow = useSyncExternalStore(subscribeViewport, narrowViewport, serverViewport);
	const [zoomOverride, setZoom] = useState<Zoom | null>(null);
	/* Bumped by every zoom request, so re-centering runs even when the scale
	   itself does not change: the same zoom, a different finding. */
	const [zoomRequest, bumpZoomRequest] = useReducer((count: number) => count + 1, 0);
	const anchorRef = useRef<ZoomAnchor | null>(null);
	const viewportRef = useRef<HTMLDivElement | null>(null);
	const svgRef = useRef<SVGSVGElement | null>(null);
	/* Set on pointerup when the press turned out to be a pan, and read by the
	   marker click handler on the click event that follows. Without it, every
	   drag that happens to start on a marker opens that finding. */
	const draggedRef = useRef(false);
	const [panning, setPanning] = useState(false);
	/* Fit width of a desktop capture on a phone draws the page at about 15%,
	   and 100% of the image is twice the page's real size. Small screens open
	   at real size on the first finding instead (see the layout effect below).
	   With nothing located, fit width at least shows the whole page, never its
	   empty top-left margin. */
	const zoom: Zoom = zoomOverride ?? (isNarrow && firstElement ? actualSize : 'fit');

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
		const first = firstElement;
		if (first && zoom !== 'fit') scrollToElement(first, 'auto');
	}, [screenshotReady, overviewUrl, firstElement, zoom, scrollToElement]);

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

	return {
		viewportRef,
		svgRef,
		draggedRef,
		panning,
		handlePointerDown,
		scrollToElement,
		zoom,
		zoomToFinding,
		fitWidth: () => requestZoom('fit', focusAnchor()),
		zoomIn: () => zoomBy(ZOOM_STEP),
		zoomOut: () => zoomBy(1 / ZOOM_STEP)
	};
}
