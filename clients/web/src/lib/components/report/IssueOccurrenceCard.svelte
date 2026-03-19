<script lang="ts">
import type { IssueDetail, PageSummary } from "$lib/types/unified-report";

import { Panel } from "$lib/components/ui";
import { getCroppedViewBox, getSeverityStrokeColor } from "$lib/report";
import { cn } from "$lib/utils";
import { extractPrimaryFailureDetail } from "$lib/utils/failure-summary";
import { Copy } from "lucide-svelte";

interface Props {
	occurrence: NonNullable<IssueDetail["occurrences"]>[number];
	index: number;
	issue: IssueDetail;
	page: PageSummary | null;
	pageOverviewUrl: string | null;
	isHighlighted: boolean;
	showDetails: boolean;
	onHighlight?: () => void;
}

const {
	occurrence,
	index,
	issue,
	page,
	pageOverviewUrl,
	isHighlighted,
	showDetails,
	onHighlight,
}: Props = $props();

const overviewEl = $derived.by(() => {
	const overview = page?.pageOverview;
	if (!overview) return null;
	const elements = overview.elements ?? [];
	return (
		elements.find((el) => el.issueId === issue.id && el.nodeIndex === index) ??
		null
	);
});

const viewBox = $derived.by(() => {
	const overview = page?.pageOverview;
	if (!overview || !overviewEl) return null;
	return getCroppedViewBox(overview.pageWidth, overview.pageHeight, overviewEl);
});

async function copyToClipboard(text: string) {
	try {
		await navigator.clipboard.writeText(text);
	} catch {
		const ta = document.createElement("textarea");
		ta.value = text;
		ta.style.position = "fixed";
		ta.style.left = "-9999px";
		document.body.appendChild(ta);
		ta.select();
		try {
			document.execCommand("copy");
		} finally {
			document.body.removeChild(ta);
		}
	}
}

function getOccurrenceSnippet(): string | null {
	const parts: string[] = [];
	if (occurrence.ancestorPath)
		parts.push(`Location: ${occurrence.ancestorPath}`);
	if (occurrence.selector) parts.push(`Selector: ${occurrence.selector}`);
	if (occurrence.contextHtml) {
		parts.push(`Context HTML:\n${occurrence.contextHtml}`);
	} else if (occurrence.html) {
		parts.push(`HTML: ${occurrence.html}`);
	}
	if (occurrence.failureSummary)
		parts.push(`Fix: ${occurrence.failureSummary}`);
	return parts.length ? parts.join("\n\n") : null;
}
</script>

<Panel
	padding="sm"
	rounded="lg"
	class={cn(
		isHighlighted ? 'border-accent bg-accent/5 ring-accent/30 ring-2' : 'bg-surface-muted'
	)}
	data-occurrence-id={occurrence.elementId ?? undefined}
>
	<div class={cn('grid gap-3', viewBox && pageOverviewUrl ? 'md:grid-cols-[220px,1fr]' : '')}>
		{#if viewBox && pageOverviewUrl && overviewEl && page?.pageOverview}
			<button
				type="button"
				class="group border-line bg-surface relative overflow-hidden rounded-md border shadow-sm"
				onclick={onHighlight}
				aria-label="Highlight this element on the page"
				title="Click to highlight this element"
			>
				<svg
					class="bg-surface-muted h-[140px] w-full"
					viewBox={`${viewBox.x} ${viewBox.y} ${viewBox.width} ${viewBox.height}`}
					preserveAspectRatio="xMidYMid slice"
					role="img"
					aria-label="Page screenshot crop"
				>
					<image
						href={pageOverviewUrl}
						x="0"
						y="0"
						width={page.pageOverview.pageWidth}
						height={page.pageOverview.pageHeight}
					/>
					<rect
						x={overviewEl.x}
						y={overviewEl.y}
						width={overviewEl.width}
						height={overviewEl.height}
						fill="transparent"
						stroke={getSeverityStrokeColor(issue.severity)}
						stroke-width="6"
					/>
				</svg>
				<div
					class="pointer-events-none absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/60 to-transparent px-2 py-1 text-left text-[11px] text-white opacity-0 transition group-hover:opacity-100"
				>
					Click to focus this occurrence
				</div>
			</button>
		{/if}

		<div class="min-w-0">
			{#if showDetails && occurrence.ancestorPath}
				<p class="text-ink-faint mb-2 font-mono text-xs" title="Element location in DOM">
					{occurrence.ancestorPath}
				</p>
			{/if}

			{#if showDetails && occurrence.selector}
				<div class="flex items-start justify-between gap-2">
					<p class="text-ink-muted mb-1 flex-1 break-all font-mono text-xs">
						{occurrence.selector}
					</p>
					<button
						onclick={() => occurrence.selector && copyToClipboard(occurrence.selector)}
						class="text-ink-muted hover:text-ink rounded p-1"
						aria-label="Copy selector"
						title="Copy selector"
					>
						<Copy class="h-4 w-4" />
					</button>
				</div>
			{/if}

			{#if !showDetails && occurrence.label}
				<p class="text-ink mt-2 text-sm font-medium line-clamp-1">{occurrence.label}</p>
			{/if}

			{#if occurrence.failureSummary}
				{@const primaryFailure = extractPrimaryFailureDetail(occurrence.failureSummary)}
				<p
					class={cn(
						'text-ink-muted mt-2 text-sm',
						showDetails ? 'whitespace-pre-wrap' : 'line-clamp-3'
					)}
				>
					{showDetails ? occurrence.failureSummary : primaryFailure ?? occurrence.failureSummary}
				</p>
			{:else if !showDetails && occurrence.textSnippet}
				<p class="text-ink-muted mt-2 text-sm">"{occurrence.textSnippet}"</p>
			{/if}

			{#if showDetails && occurrence.contextHtml}
				<details class="mt-3">
					<summary class="text-ink-muted hover:text-ink cursor-pointer text-sm">
						Show HTML context
					</summary>
					<div class="relative mt-2">
						<button
							onclick={() => occurrence.contextHtml && copyToClipboard(occurrence.contextHtml)}
							class="absolute top-2 right-2 z-10 rounded bg-neutral-700/60 p-1 text-neutral-100 hover:bg-neutral-700"
							aria-label="Copy context HTML"
							title="Copy context HTML"
						>
							<Copy class="h-4 w-4" />
						</button>
						<pre
							class="max-h-64 overflow-x-auto overflow-y-auto rounded bg-neutral-800 p-2 text-xs text-neutral-100">{occurrence.contextHtml}</pre>
						<p class="text-ink-faint mt-1 text-xs italic">
							Context: element with parent and siblings
						</p>
					</div>
				</details>
			{:else if showDetails && occurrence.html}
				<details class="mt-3">
					<summary class="text-ink-muted hover:text-ink cursor-pointer text-sm">
						Show element HTML
					</summary>
					<div class="relative mt-2">
						<button
							onclick={() => occurrence.html && copyToClipboard(occurrence.html)}
							class="absolute top-2 right-2 rounded bg-neutral-700/60 p-1 text-neutral-100 hover:bg-neutral-700"
							aria-label="Copy HTML"
							title="Copy HTML"
						>
							<Copy class="h-4 w-4" />
						</button>
						<pre
							class="overflow-x-auto rounded bg-neutral-800 p-2 text-xs text-neutral-100">{occurrence.html}</pre>
					</div>
				</details>
			{/if}

			{#if showDetails}
				<div class="mt-3 flex flex-wrap items-center gap-3">
					<button
						onclick={() => {
							const snippet = getOccurrenceSnippet();
							if (snippet) copyToClipboard(snippet);
						}}
						class="text-accent inline-flex items-center gap-2 text-sm hover:underline"
						type="button"
					>
						<Copy class="h-4 w-4" />
						Copy details
					</button>
					{#if occurrence.failureSummary}
						<button
							onclick={() => occurrence.failureSummary && copyToClipboard(occurrence.failureSummary)}
							class="text-ink-muted inline-flex items-center gap-2 text-sm hover:underline"
							type="button"
							aria-label="Copy fix guidance"
							title="Copy fix guidance"
						>
							<Copy class="h-4 w-4" />
							Copy fix
						</button>
					{/if}
				</div>
			{/if}
		</div>
	</div>
</Panel>
