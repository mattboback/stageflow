import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type { Rect, SampleSlot, ViewBox } from '../../../lib/report';
import { rgbToHex } from '../../../lib/utils/contrast';

interface Props {
	imageUrl: string;
	pageWidth: number;
	pageHeight: number;
	viewBox: ViewBox;
	element?: Rect | null;
	onPick: (slot: SampleSlot, hex: string) => void;
}

interface Point {
	x: number;
	y: number;
}

interface ImageSource {
	url: string;
	ctx: CanvasRenderingContext2D;
	pixelScale: number;
}

const LOUPE_GRID = 9;
const LOUPE_ZOOM = 12;
const LOUPE_SIZE = LOUPE_GRID * LOUPE_ZOOM;

const SLOT_LABELS: Record<SampleSlot, string> = { fg: 'Text', bg: 'Background' };

function defaultCursor(viewBox: ViewBox, element: Rect | null): Point {
	return element
		? { x: element.x + element.width / 2, y: element.y + element.height / 2 }
		: { x: viewBox.x + viewBox.width / 2, y: viewBox.y + viewBox.height / 2 };
}

function cursorKey(viewBox: ViewBox, element: Rect | null): string {
	return [viewBox.x, viewBox.y, viewBox.width, viewBox.height, element?.x, element?.y].join(
		':'
	);
}

export function ContrastSampler({
	imageUrl,
	pageWidth,
	pageHeight,
	viewBox,
	element = null,
	onPick
}: Props) {
	const [activeSlot, setActiveSlot] = useState<SampleSlot>('fg');
	const [failedUrl, setFailedUrl] = useState<string | null>(null);
	const [source, setSource] = useState<ImageSource | null>(null);

	// Cursor state is keyed to the crop, so switching issues re-centers on the
	// new element without a reset effect.
	const key = cursorKey(viewBox, element);
	const [cursorState, setCursorState] = useState<{ key: string; point: Point } | null>(
		null
	);
	const cursor = cursorState?.key === key ? cursorState.point : defaultCursor(viewBox, element);

	const svgRef = useRef<SVGSVGElement | null>(null);
	const loupeRef = useRef<HTMLCanvasElement | null>(null);

	const imageFailed = failedUrl === imageUrl;
	const activeSource = source?.url === imageUrl ? source : null;

	useEffect(() => {
		let cancelled = false;
		const img = new Image();
		img.crossOrigin = 'anonymous';
		img.onload = () => {
			if (cancelled) return;
			const canvas = document.createElement('canvas');
			canvas.width = img.naturalWidth;
			canvas.height = img.naturalHeight;
			const ctx = canvas.getContext('2d', { willReadFrequently: true });
			if (!ctx) {
				setFailedUrl(imageUrl);
				return;
			}
			ctx.drawImage(img, 0, 0);
			try {
				ctx.getImageData(0, 0, 1, 1);
			} catch {
				// Tainted canvas — cross-origin image without CORS headers.
				setFailedUrl(imageUrl);
				return;
			}
			setSource({
				url: imageUrl,
				ctx,
				pixelScale: pageWidth > 0 ? img.naturalWidth / pageWidth : 1
			});
		};
		img.onerror = () => {
			if (!cancelled) setFailedUrl(imageUrl);
		};
		img.src = imageUrl;
		return () => {
			cancelled = true;
		};
	}, [imageUrl, pageWidth]);

	const sampleAt = useCallback(
		(x: number, y: number): string | null => {
			if (!activeSource) return null;
			const { ctx, pixelScale } = activeSource;
			const px = Math.min(ctx.canvas.width - 1, Math.max(0, Math.round(x * pixelScale)));
			const py = Math.min(ctx.canvas.height - 1, Math.max(0, Math.round(y * pixelScale)));
			const [r, g, b] = ctx.getImageData(px, py, 1, 1).data;
			return rgbToHex({ r: r ?? 0, g: g ?? 0, b: b ?? 0 });
		},
		[activeSource]
	);

	const cursorHex = useMemo(
		() => sampleAt(cursor.x, cursor.y),
		[sampleAt, cursor.x, cursor.y]
	);

	// Canvas repaint is DOM synchronization, so an effect is the right tool.
	useEffect(() => {
		const loupe = loupeRef.current;
		if (!loupe || !activeSource) return;
		const ctx = loupe.getContext('2d');
		if (!ctx) return;
		const { ctx: sourceCtx, pixelScale } = activeSource;
		const px = Math.round(cursor.x * pixelScale);
		const py = Math.round(cursor.y * pixelScale);
		ctx.imageSmoothingEnabled = false;
		ctx.fillStyle = '#e9edee';
		ctx.fillRect(0, 0, LOUPE_SIZE, LOUPE_SIZE);
		ctx.drawImage(
			sourceCtx.canvas,
			px - (LOUPE_GRID - 1) / 2,
			py - (LOUPE_GRID - 1) / 2,
			LOUPE_GRID,
			LOUPE_GRID,
			0,
			0,
			LOUPE_SIZE,
			LOUPE_SIZE
		);
		const center = Math.floor(LOUPE_GRID / 2) * LOUPE_ZOOM;
		ctx.strokeStyle = '#ffffff';
		ctx.strokeRect(center - 1.5, center - 1.5, LOUPE_ZOOM + 3, LOUPE_ZOOM + 3);
		ctx.strokeStyle = '#1f2933';
		ctx.strokeRect(center - 0.5, center - 0.5, LOUPE_ZOOM + 1, LOUPE_ZOOM + 1);
	}, [activeSource, cursor.x, cursor.y]);

	const moveCursorTo = (x: number, y: number) => {
		setCursorState({
			key,
			point: {
				x: Math.min(viewBox.x + viewBox.width, Math.max(viewBox.x, x)),
				y: Math.min(viewBox.y + viewBox.height, Math.max(viewBox.y, y))
			}
		});
	};

	const handlePointerMove = (event: React.PointerEvent<SVGSVGElement>) => {
		const ctm = svgRef.current?.getScreenCTM();
		if (!ctm) return;
		const point = new DOMPoint(event.clientX, event.clientY).matrixTransform(
			ctm.inverse()
		);
		moveCursorTo(point.x, point.y);
	};

	const pick = () => {
		const hex = sampleAt(cursor.x, cursor.y);
		if (!hex) return;
		onPick(activeSlot, hex);
		if (activeSlot === 'fg') setActiveSlot('bg');
	};

	const handleKeydown = (event: React.KeyboardEvent<HTMLDivElement>) => {
		const step = event.shiftKey ? 10 : 1;
		const moves: Record<string, [number, number]> = {
			ArrowLeft: [-step, 0],
			ArrowRight: [step, 0],
			ArrowUp: [0, -step],
			ArrowDown: [0, step]
		};
		const move = moves[event.key];
		if (move) {
			event.preventDefault();
			moveCursorTo(cursor.x + move[0], cursor.y + move[1]);
		} else if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			pick();
		}
	};

	if (imageFailed) {
		return (
			<p className="vfy__notice">
				The screenshot could not be loaded for sampling. Enter the colors manually
				below.
			</p>
		);
	}

	return (
		<div className="vfy__sampler">
			<div className="vfy__sampler-head">
				<div className="vfy__slots" role="group" aria-label="Color to sample next">
					{(['fg', 'bg'] as const).map((slot) => (
						<button
							key={slot}
							type="button"
							aria-pressed={activeSlot === slot}
							className="vfy__slot-btn"
							onClick={() => setActiveSlot(slot)}
						>
							{SLOT_LABELS[slot]}
						</button>
					))}
				</div>
				<p className="vfy__hint">
					Click the image to sample the {SLOT_LABELS[activeSlot].toLowerCase()} color.
					Sample the thickest part of a letter stroke — edges are anti-aliased.
				</p>
			</div>
			<div className="vfy__stage">
				{/* role="application" scopes arrow keys to the sampler widget */}
				<div
					role="application"
					tabIndex={0}
					aria-label="Color sampler. Use arrow keys to move the sampling point, Enter to sample the color under it."
					className="vfy__stage-focus"
					onKeyDown={handleKeydown}
				>
					<svg
						ref={svgRef}
						className="vfy__svg"
						viewBox={`${viewBox.x} ${viewBox.y} ${viewBox.width} ${viewBox.height}`}
						preserveAspectRatio="xMidYMid meet"
						aria-hidden="true"
						onPointerMove={handlePointerMove}
						onClick={pick}
					>
						<image href={imageUrl} x={0} y={0} width={pageWidth} height={pageHeight} />
						{element && (
							<rect
								x={element.x}
								y={element.y}
								width={element.width}
								height={element.height}
								fill="none"
								stroke="#1f6f73"
								strokeWidth={2}
								strokeDasharray="6 4"
								vectorEffect="non-scaling-stroke"
							/>
						)}
						<circle
							cx={cursor.x}
							cy={cursor.y}
							r={7}
							fill="none"
							stroke="#ffffff"
							strokeWidth={3}
							vectorEffect="non-scaling-stroke"
						/>
						<circle
							cx={cursor.x}
							cy={cursor.y}
							r={7}
							fill="none"
							stroke="#1f2933"
							strokeWidth={1}
							vectorEffect="non-scaling-stroke"
						/>
					</svg>
				</div>
			</div>

			<div className="vfy__sampler-foot">
				<canvas
					ref={loupeRef}
					width={LOUPE_SIZE}
					height={LOUPE_SIZE}
					className="vfy__loupe"
					aria-hidden="true"
				/>
				<span className="vfy__cursor-hex mono">{cursorHex ?? '—'}</span>
			</div>
		</div>
	);
}
