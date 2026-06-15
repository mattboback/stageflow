<script lang="ts">
	import type { Rect, SampleSlot, ViewBox } from '$lib/report';

	import { rgbToHex } from '$lib/utils/contrast';

	interface Props {
		imageUrl: string;
		pageWidth: number;
		pageHeight: number;
		viewBox: ViewBox;
		element?: Rect | null;
		onPick: (slot: SampleSlot, hex: string) => void;
	}

	let { imageUrl, pageWidth, pageHeight, viewBox, element = null, onPick }: Props = $props();

	const LOUPE_GRID = 9;
	const LOUPE_ZOOM = 12;
	const LOUPE_SIZE = LOUPE_GRID * LOUPE_ZOOM;

	let activeSlot = $state<SampleSlot>('fg');
	let imageFailed = $state(false);
	let imageReady = $state(false);
	let cursorHex = $state<string | null>(null);
	let cursor = $state({ x: 0, y: 0 });

	let svgEl = $state<SVGSVGElement | null>(null);
	let loupeEl = $state<HTMLCanvasElement | null>(null);
	let sourceCtx: CanvasRenderingContext2D | null = null;
	let pixelScale = 1;

	$effect(() => {
		cursor = element
			? { x: element.x + element.width / 2, y: element.y + element.height / 2 }
			: { x: viewBox.x + viewBox.width / 2, y: viewBox.y + viewBox.height / 2 };
	});

	$effect(() => {
		imageReady = false;
		imageFailed = false;
		sourceCtx = null;
		const img = new Image();
		img.crossOrigin = 'anonymous';
		img.onload = () => {
			const canvas = document.createElement('canvas');
			canvas.width = img.naturalWidth;
			canvas.height = img.naturalHeight;
			const ctx = canvas.getContext('2d', { willReadFrequently: true });
			if (!ctx) {
				imageFailed = true;
				return;
			}
			ctx.drawImage(img, 0, 0);
			pixelScale = pageWidth > 0 ? img.naturalWidth / pageWidth : 1;
			try {
				ctx.getImageData(0, 0, 1, 1);
			} catch {
				imageFailed = true;
				return;
			}
			sourceCtx = ctx;
			imageReady = true;
			refreshCursor();
		};
		img.onerror = () => {
			imageFailed = true;
		};
		img.src = imageUrl;
	});

	function sampleAt(x: number, y: number): string | null {
		if (!sourceCtx) return null;
		const px = Math.min(sourceCtx.canvas.width - 1, Math.max(0, Math.round(x * pixelScale)));
		const py = Math.min(sourceCtx.canvas.height - 1, Math.max(0, Math.round(y * pixelScale)));
		const [r, g, b] = sourceCtx.getImageData(px, py, 1, 1).data;
		return rgbToHex({ r, g, b });
	}

	function drawLoupe() {
		if (!loupeEl || !sourceCtx) return;
		const ctx = loupeEl.getContext('2d');
		if (!ctx) return;
		const px = Math.round(cursor.x * pixelScale);
		const py = Math.round(cursor.y * pixelScale);
		ctx.imageSmoothingEnabled = false;
		ctx.fillStyle = '#e8e5df';
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
		ctx.strokeStyle = '#1a1714';
		ctx.strokeRect(center - 0.5, center - 0.5, LOUPE_ZOOM + 1, LOUPE_ZOOM + 1);
	}

	function refreshCursor() {
		cursorHex = sampleAt(cursor.x, cursor.y);
		drawLoupe();
	}

	function moveCursorTo(x: number, y: number) {
		cursor = {
			x: Math.min(viewBox.x + viewBox.width, Math.max(viewBox.x, x)),
			y: Math.min(viewBox.y + viewBox.height, Math.max(viewBox.y, y))
		};
		refreshCursor();
	}

	function handlePointerMove(event: PointerEvent) {
		const ctm = svgEl?.getScreenCTM();
		if (!ctm) return;
		const point = new DOMPoint(event.clientX, event.clientY).matrixTransform(ctm.inverse());
		moveCursorTo(point.x, point.y);
	}

	function pick() {
		const hex = sampleAt(cursor.x, cursor.y);
		if (!hex) return;
		onPick(activeSlot, hex);
		if (activeSlot === 'fg') activeSlot = 'bg';
	}

	function handleKeydown(event: KeyboardEvent) {
		const step = event.shiftKey ? 10 : 1;
		const moves: Record<string, [number, number]> = {
			ArrowLeft: [-step, 0],
			ArrowRight: [step, 0],
			ArrowUp: [0, -step],
			ArrowDown: [0, step]
		};
		if (moves[event.key]) {
			event.preventDefault();
			moveCursorTo(cursor.x + moves[event.key][0], cursor.y + moves[event.key][1]);
		} else if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			pick();
		}
	}

	const slotLabels: Record<SampleSlot, string> = { fg: 'Text', bg: 'Background' };
</script>

{#if imageFailed}
	<p class="border-line bg-surface-muted/40 text-ink-muted rounded-md border p-3 text-sm">
		The screenshot could not be loaded for sampling. Enter the colors manually below.
	</p>
{:else}
	<div class="flex flex-wrap gap-4">
		<div class="min-w-0 flex-1 basis-72">
			<!-- role="application" is the correct pattern for a composite widget that owns
			     arrow-key handling; Svelte's checker doesn't recognise it as interactive. -->
			<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
			<div
				role="application"
				tabindex="0"
				aria-label="Color sampler. Use arrow keys to move the sampling point, Enter to sample the color under it."
				class="border-line focus-visible:ring-accent rounded-md border focus-visible:ring-2 focus-visible:outline-none"
				onkeydown={handleKeydown}
			>
				<svg
					bind:this={svgEl}
					class="bg-surface-muted block max-h-[360px] w-full cursor-crosshair rounded-md"
					viewBox={`${viewBox.x} ${viewBox.y} ${viewBox.width} ${viewBox.height}`}
					preserveAspectRatio="xMidYMid meet"
					aria-hidden="true"
					onpointermove={handlePointerMove}
					onclick={pick}
				>
					<image href={imageUrl} x="0" y="0" width={pageWidth} height={pageHeight} />
					{#if element}
						<rect
							x={element.x}
							y={element.y}
							width={element.width}
							height={element.height}
							fill="none"
							stroke="#1b5c5e"
							stroke-width="2"
							stroke-dasharray="6 4"
							vector-effect="non-scaling-stroke"
						/>
					{/if}
					<circle
						cx={cursor.x}
						cy={cursor.y}
						r="7"
						fill="none"
						stroke="#ffffff"
						stroke-width="3"
						vector-effect="non-scaling-stroke"
					/>
					<circle
						cx={cursor.x}
						cy={cursor.y}
						r="7"
						fill="none"
						stroke="#1a1714"
						stroke-width="1"
						vector-effect="non-scaling-stroke"
					/>
				</svg>
			</div>
			<p class="text-ink-muted mt-2 text-xs">
				Click the image to sample the {slotLabels[activeSlot].toLowerCase()} color. Sample the thickest
				part of a letter stroke — edges are anti-aliased.
			</p>
		</div>

		<div class="flex w-32 shrink-0 flex-col items-center gap-2">
			<div class="flex w-full gap-1" role="group" aria-label="Color to sample next">
				{#each ['fg', 'bg'] as const as slot (slot)}
					<button
						type="button"
						aria-pressed={activeSlot === slot}
						class={[
							'flex-1 rounded-sm border px-1.5 py-1 text-[11px] font-semibold transition-colors',
							activeSlot === slot
								? 'border-accent bg-accent-soft text-accent-ink'
								: 'border-line text-ink-muted hover:text-ink bg-surface'
						].join(' ')}
						onclick={() => (activeSlot = slot)}
					>
						{slotLabels[slot]}
					</button>
				{/each}
			</div>
			<canvas
				bind:this={loupeEl}
				width={LOUPE_SIZE}
				height={LOUPE_SIZE}
				class="border-line rounded-md border"
				aria-hidden="true"
			></canvas>
			<p class="stat-mono text-ink text-xs" aria-live="polite">
				{imageReady && cursorHex ? cursorHex : '······'}
			</p>
		</div>
	</div>
{/if}
