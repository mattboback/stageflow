<script lang="ts">
import { Modal } from "$lib/components/ui";
import { cn } from "$lib/utils";
import { onMount } from "svelte";

interface Props {
	chart: string;
	class?: string;
}

let { chart, class: className = "" }: Props = $props();

let containerRef = $state<HTMLDivElement | null>(null);
let svg = $state("");
const svgDataUrl = $derived.by(() => {
	if (!svg) return "";
	return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
});
let error = $state("");
let isLoading = $state(true);
let isFullscreen = $state(false);
let renderCount = 0;

async function renderDiagram(chartData: string, diagramId: string) {
	const mermaid = (await import("mermaid")).default;

	mermaid.initialize({
		startOnLoad: false,
		theme: "base",
		themeVariables: {
			primaryColor: "#FDE8E4",
			primaryTextColor: "#111111",
			primaryBorderColor: "#E23D28",
			lineColor: "#4B5563",
			secondaryColor: "#FBFAF7",
			tertiaryColor: "#FFF3E0",
			background: "#FFFFFF",
			mainBkg: "#FFFFFF",
			secondBkg: "#FBFAF7",
			clusterBkg: "#FBFAF7",
			clusterBorder: "#E6E2D8",
			fontSize: "15px",
			fontFamily:
				'"Source Sans 3 Variable", ui-sans-serif, system-ui, -apple-system, sans-serif',
			edgeLabelBackground: "#FFFFFF",
			tertiaryTextColor: "#111111",
			textColor: "#111111",
			nodeBorder: "#E6E2D8",
			titleColor: "#111111",
		},
		flowchart: {
			curve: "basis",
			padding: 24,
			nodeSpacing: 60,
			rankSpacing: 60,
			diagramPadding: 16,
			useMaxWidth: true,
		},
		sequence: {
			diagramMarginX: 24,
			diagramMarginY: 24,
			actorMargin: 80,
			width: 200,
			height: 70,
			boxMargin: 12,
			boxTextMargin: 8,
			noteMargin: 12,
			messageMargin: 50,
			useMaxWidth: true,
		},
	});

	try {
		const { svg: renderedSvg } = await mermaid.render(diagramId, chartData);
		return { svg: renderedSvg, error: "" };
	} catch (err) {
		console.error("Mermaid rendering error:", err);
		return {
			svg: "",
			error: err instanceof Error ? err.message : "Failed to render diagram",
		};
	}
}

onMount(() => {
	if (chart) {
		renderCount += 1;
		const id = `mermaid-${renderCount}-${Date.now()}`;

		void renderDiagram(chart, id).then((result) => {
			svg = result.svg;
			error = result.error;
			isLoading = false;
		});
	} else {
		isLoading = false;
		error = "No chart data provided";
	}
});

function openFullscreen() {
	isFullscreen = true;
}

function closeFullscreen() {
	isFullscreen = false;
}
</script>

<svelte:boundary onerror={(e) => console.error('Mermaid diagram render error:', e)}>
	{#snippet failed(error)}
		{@const errorMessage = error instanceof Error ? error.message : 'Unexpected error'}
		<div
			class="flex items-center justify-center rounded-lg border border-red-200 bg-red-50 p-8 text-center"
		>
			<div>
				<p class="mb-2 text-sm font-semibold text-red-900">Failed to render diagram</p>
				<p class="text-xs text-red-700">{errorMessage}</p>
			</div>
		</div>
	{/snippet}
	{#if !chart}
		<div
			class="flex items-center justify-center rounded-lg border border-red-200 bg-red-50 p-8 text-center"
		>
			<div>
				<p class="mb-2 text-sm font-semibold text-red-900">Failed to render diagram</p>
				<p class="text-xs text-red-700">No chart data provided</p>
			</div>
		</div>
	{:else if error}
		<div
			class="flex items-center justify-center rounded-lg border border-red-200 bg-red-50 p-8 text-center"
		>
			<div>
				<p class="mb-2 text-sm font-semibold text-red-900">Failed to render diagram</p>
				<p class="text-xs text-red-700">{error}</p>
			</div>
		</div>
	{:else if isLoading}
		<div class="border-line bg-surface-muted flex items-center justify-center rounded-lg border p-12">
			<div class="text-center">
				<div
					class="border-line border-t-accent mb-3 inline-block h-8 w-8 animate-spin rounded-full border-4"
				></div>
				<p class="text-ink-faint text-sm">Rendering diagram...</p>
			</div>
		</div>
	{:else}
		<div class="relative">
			<button
				onclick={openFullscreen}
				class="bg-surface/90 text-ink-muted hover:bg-surface hover:text-ink absolute top-2 right-2 z-10 rounded-lg p-2 shadow-md transition-all hover:shadow-lg"
				aria-label="View fullscreen"
				title="View fullscreen"
			>
				<svg
					xmlns="http://www.w3.org/2000/svg"
					class="h-5 w-5"
					fill="none"
					viewBox="0 0 24 24"
					stroke="currentColor"
				>
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5l-5-5m5 5v-4m0 4h-4"
					/>
				</svg>
			</button>
			<div bind:this={containerRef} class={cn('mermaid-container overflow-x-auto', className)}>
				{#if svgDataUrl}
					<img class="block h-auto max-w-none" src={svgDataUrl} alt="Mermaid diagram" />
				{/if}
			</div>
		</div>

		{#if isFullscreen}
			<Modal
				open={true}
				onClose={closeFullscreen}
				ariaLabel="Fullscreen diagram"
				backdrop="dark"
				size="auto"
				contentClass="bg-surface relative max-h-full max-w-full overflow-auto rounded-lg p-8"
			>
				<button
					data-modal-initial-focus
					onclick={closeFullscreen}
					class="bg-surface-muted text-ink-muted hover:bg-surface absolute top-4 right-4 rounded-lg p-2 transition-colors"
					aria-label="Close fullscreen"
				>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						class="h-6 w-6"
						fill="none"
						viewBox="0 0 24 24"
						stroke="currentColor"
					>
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				</button>
				<div class="mermaid-fullscreen">
					{#if svgDataUrl}
						<img class="block h-auto max-w-none" src={svgDataUrl} alt="Mermaid diagram" />
					{/if}
				</div>
			</Modal>
		{/if}
	{/if}
</svelte:boundary>
