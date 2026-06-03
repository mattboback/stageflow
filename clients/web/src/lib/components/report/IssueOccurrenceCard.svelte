<script lang="ts">
	import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';
	import { getCroppedViewBox, getSeverityStrokeColor } from '$lib/report';
	import { cn } from '$lib/utils';
	import { extractPrimaryFailureDetail } from '$lib/utils/failure-summary';
	import { Copy } from 'lucide-svelte';
	import { slide } from 'svelte/transition';

	interface Props {
		occurrence: NonNullable<IssueDetail['occurrences']>[number];
		index: number;
		issue: IssueDetail;
		page: PageSummary | null;
		pageOverviewUrl: string | null;
		isHighlighted: boolean;
		showDetails: boolean;
		hideThumbnail?: boolean;
		onHighlight?: () => void;
	}

	let {
		occurrence,
		index,
		issue,
		page,
		pageOverviewUrl,
		isHighlighted,
		showDetails,
		hideThumbnail = false,
		onHighlight
	}: Props = $props();

	let isHtmlExpanded = $state(false);
	let copiedType = $state<'html' | 'selector' | 'details' | 'fix' | null>(null);

	const overviewEl = $derived.by(() => {
		const overview = page?.pageOverview;
		if (!overview) return null;
		const elements = overview.elements ?? [];
		return elements.find((el) => el.issueId === issue.id && el.nodeIndex === index) ?? null;
	});

	const viewBox = $derived.by(() => {
		const overview = page?.pageOverview;
		if (!overview || !overviewEl) return null;
		return getCroppedViewBox(overview.pageWidth, overview.pageHeight, overviewEl, {
			padding: 24,
			minWidth: 280,
			minHeight: 180
		});
	});

	async function copyToClipboard(text: string) {
		try {
			await navigator.clipboard.writeText(text);
		} catch {
			const ta = document.createElement('textarea');
			ta.value = text;
			ta.style.position = 'fixed';
			ta.style.left = '-9999px';
			document.body.appendChild(ta);
			ta.select();
			try {
				document.execCommand('copy');
			} finally {
				document.body.removeChild(ta);
			}
		}
	}

	async function handleCopy(text: string, type: 'html' | 'selector' | 'details' | 'fix') {
		await copyToClipboard(text);
		copiedType = type;
		setTimeout(() => {
			if (copiedType === type) copiedType = null;
		}, 1500);
	}

	function getOccurrenceSnippet(): string | null {
		const parts: string[] = [];
		if (occurrence.ancestorPath) parts.push(`Location: ${occurrence.ancestorPath}`);
		if (occurrence.selector) parts.push(`Selector: ${occurrence.selector}`);
		if (occurrence.contextHtml) {
			parts.push(`Context HTML:\n${occurrence.contextHtml}`);
		} else if (occurrence.html) {
			parts.push(`HTML: ${occurrence.html}`);
		}
		if (occurrence.failureSummary) parts.push(`Fix: ${occurrence.failureSummary}`);
		return parts.length ? parts.join('\n\n') : null;
	}
</script>

<Panel
	padding="sm"
	rounded="lg"
	class={cn(
		'transition-all duration-300',
		isHighlighted
			? 'border-accent bg-accent/5 ring-accent/30 scale-[1.005] shadow-sm ring-2'
			: 'bg-surface-muted border-line hover:border-accent/25 hover:shadow-xs'
	)}
	data-occurrence-id={occurrence.elementId ?? undefined}
>
	<div
		class={cn(
			'grid gap-4',
			!hideThumbnail && viewBox && pageOverviewUrl ? 'md:grid-cols-[220px_1fr]' : ''
		)}
	>
		{#if !hideThumbnail && viewBox && pageOverviewUrl && overviewEl && page?.pageOverview}
			<button
				type="button"
				class="group border-line bg-surface relative overflow-hidden rounded-xl border shadow-xs transition-transform duration-200 hover:scale-[1.01]"
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
					class="pointer-events-none absolute inset-x-0 bottom-0 bg-gradient-to-t from-black/80 to-transparent px-3 py-1.5 text-left text-[11px] font-semibold text-white opacity-0 transition group-hover:opacity-100"
				>
					Click to focus this occurrence
				</div>
			</button>
		{/if}

		<div class="flex min-w-0 flex-col justify-between">
			<div>
				{#if showDetails && occurrence.ancestorPath}
					<p
						class="text-ink-faint mb-2 font-mono text-[10px] tracking-wider uppercase"
						title="Element location in DOM"
					>
						{occurrence.ancestorPath}
					</p>
				{/if}

				{#if showDetails && occurrence.selector}
					<div
						class="border-line bg-surface/50 mb-3 flex items-center justify-between gap-3 rounded-xl border p-2.5 shadow-inner"
					>
						<p class="text-ink-muted flex-1 font-mono text-xs leading-normal break-all">
							{occurrence.selector}
						</p>
						<button
							onclick={() => occurrence.selector && handleCopy(occurrence.selector, 'selector')}
							class="text-ink-muted hover:text-accent flex shrink-0 items-center p-1.5 transition-colors"
							aria-label="Copy selector"
							title="Copy selector"
						>
							{#if copiedType === 'selector'}
								<span class="animate-pulse text-[10px] font-extrabold text-emerald-600"
									>Copied!</span
								>
							{:else}
								<Copy class="h-3.5 w-3.5" />
							{/if}
						</button>
					</div>
				{/if}

				{#if !showDetails && occurrence.label}
					<p class="text-ink mt-2 line-clamp-1 text-sm font-semibold">{occurrence.label}</p>
				{/if}

				{#if occurrence.failureSummary}
					{@const primaryFailure = extractPrimaryFailureDetail(occurrence.failureSummary)}
					<p
						class={cn(
							'text-ink-muted border-line mt-2 border-l-2 py-0.5 pl-3 text-sm leading-relaxed',
							showDetails ? 'whitespace-pre-wrap' : 'line-clamp-3'
						)}
					>
						{showDetails
							? occurrence.failureSummary
							: (primaryFailure ?? occurrence.failureSummary)}
					</p>
				{:else if !showDetails && occurrence.textSnippet}
					<p class="text-ink-muted mt-2 text-sm italic">"{occurrence.textSnippet}"</p>
				{/if}

				<!-- Smooth custom reveal drawers for HTML contexts -->
				{#if showDetails && (occurrence.contextHtml || occurrence.html)}
					{@const htmlContent = occurrence.contextHtml || occurrence.html}
					<div class="mt-4">
						<button
							type="button"
							onclick={() => (isHtmlExpanded = !isHtmlExpanded)}
							class="text-ink-muted hover:text-accent flex items-center gap-1.5 text-xs font-semibold tracking-wide uppercase transition-colors"
						>
							<span
								class="inline-block transition-transform duration-200"
								class:rotate-90={isHtmlExpanded}>▶</span
							>
							{occurrence.contextHtml ? 'HTML Context' : 'Element HTML'}
						</button>
						{#if isHtmlExpanded}
							<div transition:slide={{ duration: 250 }} class="relative mt-2">
								<button
									onclick={() => handleCopy(htmlContent!, 'html')}
									class="absolute top-2.5 right-2.5 z-10 flex items-center gap-1 rounded-lg border border-neutral-700/50 bg-neutral-900/80 px-2.5 py-1 text-[10px] font-bold text-neutral-300 backdrop-blur-md transition-all hover:bg-neutral-800"
								>
									{#if copiedType === 'html'}
										<span class="animate-pulse font-extrabold text-emerald-400">✓ Copied!</span>
									{:else}
										<Copy class="h-3 w-3" />
										Copy
									{/if}
								</button>
								<pre
									class="max-h-64 overflow-x-auto overflow-y-auto rounded-xl border border-neutral-800 bg-neutral-950 p-3.5 font-mono text-xs text-neutral-200 shadow-inner">{htmlContent}</pre>
								{#if occurrence.contextHtml}
									<p class="text-ink-faint mt-1.5 text-[10px] italic">
										Context: target element displayed with immediate DOM siblings and parent nodes.
									</p>
								{/if}
							</div>
						{/if}
					</div>
				{/if}
			</div>

			{#if showDetails}
				<div class="border-line/45 mt-4 flex flex-wrap items-center gap-3 border-t pt-3">
					<button
						onclick={() => {
							const snippet = getOccurrenceSnippet();
							if (snippet) handleCopy(snippet, 'details');
						}}
						class="text-accent inline-flex items-center gap-1.5 text-xs font-semibold transition-all hover:translate-x-0.5"
						type="button"
					>
						<Copy class="h-3.5 w-3.5" />
						{#if copiedType === 'details'}
							<span class="font-bold text-emerald-600">Details copied!</span>
						{:else}
							Copy details
						{/if}
					</button>
					{#if occurrence.failureSummary}
						<button
							onclick={() =>
								occurrence.failureSummary && handleCopy(occurrence.failureSummary, 'fix')}
							class="text-ink-muted inline-flex items-center gap-1.5 text-xs font-semibold transition-all hover:translate-x-0.5"
							type="button"
							aria-label="Copy fix guidance"
							title="Copy fix guidance"
						>
							<Copy class="h-3.5 w-3.5" />
							{#if copiedType === 'fix'}
								<span class="font-bold text-emerald-600">Fix copied!</span>
							{:else}
								Copy fix
							{/if}
						</button>
					{/if}
				</div>
			{/if}
		</div>
	</div>
</Panel>
