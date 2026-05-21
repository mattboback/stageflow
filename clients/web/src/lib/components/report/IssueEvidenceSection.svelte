<script lang="ts">
	import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';
	import {
		getCroppedViewBox,
		getIssueKind,
		getSeverityStrokeColor,
		isManualReviewIssue
	} from '$lib/report';
	import { ExternalLink, FileText, Info } from 'lucide-svelte';

	interface Props {
		issue: IssueDetail;
		page: PageSummary | null;
		screenshotUrl: string | null;
		pageOverviewUrl: string | null;
		showPageOverview?: boolean;
		hideScreenshot?: boolean;
		onElementClick?: (elementId: string) => void;
	}

	let {
		issue,
		page,
		screenshotUrl,
		pageOverviewUrl,
		showPageOverview = true,
		hideScreenshot = false,
		onElementClick
	}: Props = $props();

	const kind = $derived(getIssueKind(issue));
	const isManual = $derived(isManualReviewIssue(issue));

	const pageOverviewElements = $derived.by(() => {
		const elements = page?.pageOverview?.elements ?? [];
		return elements.filter((el) => el.issueId === issue.id);
	});

	const pageWidth = $derived(page?.pageOverview?.pageWidth ?? 0);
	const pageHeight = $derived(page?.pageOverview?.pageHeight ?? 0);
	const strokeColor = $derived(getSeverityStrokeColor(issue.severity));

	const overlayRenderable = $derived(
		!!page &&
			!!pageOverviewUrl &&
			pageWidth > 0 &&
			pageHeight > 0 &&
			pageOverviewElements.length > 0
	);

	const primaryOccurrence = $derived(issue.occurrences?.[0] ?? null);
	const primaryOverviewElement = $derived.by(() => {
		const overview = page?.pageOverview;
		if (!overview || !primaryOccurrence) return null;
		const elements = overview.elements ?? [];
		return (
			elements.find((el) => el.issueId === issue.id && el.nodeIndex === 0) ??
			elements.find((el) => el.issueId === issue.id) ??
			null
		);
	});

	const cropViewBox = $derived.by(() => {
		const overview = page?.pageOverview;
		if (!overview || !primaryOverviewElement) return null;
		return getCroppedViewBox(overview.pageWidth, overview.pageHeight, primaryOverviewElement, {
			padding: 48,
			minWidth: 480,
			minHeight: 280
		});
	});

	const showCropPanel = $derived(
		!hideScreenshot && (screenshotUrl || (cropViewBox && pageOverviewUrl && primaryOverviewElement))
	);

	const showFullPagePanel = $derived(showPageOverview && overlayRenderable);

	const pageLabel = $derived(page?.path ?? page?.url ?? null);
</script>

<div class="space-y-4">
	<div class="flex items-center justify-between gap-2">
		<h3 class="text-ink font-semibold">Evidence</h3>
		<span class="text-ink-faint font-mono text-[10px] tracking-wider uppercase">
			{kind}
		</span>
	</div>

	{#if isManual}
		<Panel variant="muted" padding="md" rounded="lg">
			<div class="flex items-start gap-3">
				<Info class="text-ink-muted mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
				<div class="space-y-1.5 text-sm">
					<p class="text-ink font-semibold">Manual review item</p>
					<p class="text-ink-muted leading-relaxed">
						{issue.scanner === 'lighthouse' ? 'Lighthouse' : issue.scanner} reported this as an informational
						check. No concrete DOM occurrence was captured — verify on the page directly using the rule
						guidance.
					</p>
					{#if pageLabel}
						<p class="text-ink-muted text-xs">
							Page: <span class="text-ink font-mono">{pageLabel}</span>
						</p>
					{/if}
				</div>
			</div>
		</Panel>
	{/if}

	{#if showCropPanel}
		<div>
			<p class="text-ink-muted mb-2 text-sm">
				{cropViewBox && primaryOverviewElement
					? 'Element in context'
					: 'Scanner-captured screenshot'}
			</p>
			<Panel padding="none" rounded="lg" class="overflow-hidden">
				{#if cropViewBox && primaryOverviewElement && pageOverviewUrl && page?.pageOverview}
					<svg
						class="bg-surface-muted block max-h-[420px] w-full"
						viewBox={`${cropViewBox.x} ${cropViewBox.y} ${cropViewBox.width} ${cropViewBox.height}`}
						preserveAspectRatio="xMidYMid meet"
						role="img"
						aria-label="Element screenshot"
					>
						<image
							href={pageOverviewUrl}
							x="0"
							y="0"
							width={page.pageOverview.pageWidth}
							height={page.pageOverview.pageHeight}
						/>
						<rect
							x={primaryOverviewElement.x}
							y={primaryOverviewElement.y}
							width={primaryOverviewElement.width}
							height={primaryOverviewElement.height}
							fill="transparent"
							stroke={strokeColor}
							stroke-width="6"
						/>
					</svg>
				{:else if screenshotUrl}
					<img
						src={screenshotUrl}
						alt="Issue highlighted on page"
						class="block max-h-[420px] w-full object-contain"
						loading="lazy"
					/>
				{/if}
			</Panel>
		</div>
	{/if}

	{#if primaryOccurrence?.selector || primaryOccurrence?.ancestorPath}
		<div class="grid gap-2 sm:grid-cols-2">
			{#if primaryOccurrence?.ancestorPath}
				<div class="border-line bg-surface-muted/50 rounded-lg border p-2.5">
					<p class="text-ink-faint mb-1 font-mono text-[10px] tracking-wider uppercase">DOM path</p>
					<p class="text-ink-muted font-mono text-xs leading-snug break-all">
						{primaryOccurrence.ancestorPath}
					</p>
				</div>
			{/if}
			{#if primaryOccurrence?.selector}
				<div class="border-line bg-surface-muted/50 rounded-lg border p-2.5">
					<p class="text-ink-faint mb-1 font-mono text-[10px] tracking-wider uppercase">Selector</p>
					<p class="text-ink-muted font-mono text-xs leading-snug break-all">
						{primaryOccurrence.selector}
					</p>
				</div>
			{/if}
		</div>
	{/if}

	{#if primaryOccurrence?.contextHtml || primaryOccurrence?.html}
		{@const html = primaryOccurrence.contextHtml || primaryOccurrence.html}
		<div>
			<p class="text-ink-muted mb-2 text-sm">
				{primaryOccurrence.contextHtml ? 'HTML context' : 'Element HTML'}
			</p>
			<pre
				class="border-line bg-surface-muted text-ink max-h-48 overflow-x-auto overflow-y-auto rounded-lg border p-3 font-mono text-xs leading-relaxed">{html}</pre>
		</div>
	{/if}

	{#if primaryOccurrence?.failureSummary}
		<div>
			<p class="text-ink-muted mb-2 text-sm">Scanner message</p>
			<p
				class="border-line bg-surface-muted/40 text-ink-muted rounded-lg border p-3 text-sm leading-relaxed whitespace-pre-wrap"
			>
				{primaryOccurrence.failureSummary}
			</p>
		</div>
	{/if}

	{#if showFullPagePanel}
		<div>
			<p class="text-ink-muted mb-2 text-sm">On the page</p>
			<Panel padding="none" rounded="lg" class="relative overflow-hidden">
				<div class="overflow-auto">
					<svg
						class="block w-full"
						viewBox={`0 0 ${pageWidth} ${pageHeight}`}
						preserveAspectRatio="xMinYMin meet"
						role="img"
						aria-label={`Page overview showing ${pageOverviewElements.length} highlighted element${pageOverviewElements.length === 1 ? '' : 's'}`}
					>
						<image href={pageOverviewUrl} x="0" y="0" width={pageWidth} height={pageHeight} />
						{#each pageOverviewElements as element (`${element.issueId}:${element.nodeIndex}`)}
							<rect
								x={element.x}
								y={element.y}
								width={element.width}
								height={element.height}
								fill="rgba(0,0,0,0)"
								stroke={strokeColor}
								stroke-width="4"
								class="cursor-pointer hover:fill-blue-500/10"
								role="button"
								tabindex="0"
								aria-label={`Highlight occurrence ${element.nodeIndex + 1}`}
								onclick={() => onElementClick?.(`${issue.id}-el-${element.nodeIndex}`)}
								onkeydown={(e) => {
									if (e.key === 'Enter' || e.key === ' ') {
										e.preventDefault();
										onElementClick?.(`${issue.id}-el-${element.nodeIndex}`);
									}
								}}
							>
								<title>Click to focus occurrence {element.nodeIndex + 1}</title>
							</rect>
						{/each}
					</svg>
				</div>
			</Panel>
			<p class="text-ink-muted mt-2 text-xs">
				Click a highlight box to jump to the matching element details below.
			</p>
		</div>
	{/if}

	{#if !showCropPanel && !primaryOccurrence?.selector && !primaryOccurrence?.contextHtml && !primaryOccurrence?.html && !primaryOccurrence?.failureSummary && !showFullPagePanel}
		{#if !isManual}
			<Panel variant="muted" padding="md" rounded="lg">
				<div class="flex items-start gap-3">
					<FileText class="text-ink-muted mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
					<div class="space-y-1 text-sm">
						<p class="text-ink font-semibold">No DOM evidence captured</p>
						<p class="text-ink-muted leading-relaxed">
							{issue.scanner} reported this issue without a concrete element target. Use the rule documentation
							to verify the finding on the live page.
						</p>
					</div>
				</div>
			</Panel>
		{/if}
		{#if issue.helpUrl}
			<a
				href={issue.helpUrl}
				target="_blank"
				rel="noopener noreferrer"
				class="text-accent inline-flex items-center gap-2 text-sm hover:underline"
			>
				<FileText class="h-4 w-4" />
				Rule documentation
				<ExternalLink class="h-3 w-3" />
			</a>
		{/if}
	{/if}
</div>
