<script lang="ts">
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

	import {
		getCroppedViewBox,
		getIssueScreenshotUrl,
		getPageOverviewUrl,
		getSeverityBadgeClass,
		getSeverityStrokeColor
	} from '$lib/report';
	import { cn } from '$lib/utils';

	interface Props {
		issue: IssueDetail;
		page: PageSummary | null;
		screenshots: ScreenshotArtifact[];
		showScreenshot?: boolean;
		isVirtualized?: boolean;
		isSelected?: boolean;
		onclick?: () => void;
	}

	const {
		issue,
		page = null,
		screenshots,
		showScreenshot = true,
		isVirtualized = false,
		isSelected = false,
		onclick
	}: Props = $props();

	const overview = $derived(page?.pageOverview ?? null);
	const overviewEl = $derived(overview?.elements?.find((el) => el.issueId === issue.id) ?? null);
	const overviewUrl = $derived(
		showScreenshot && page
			? getPageOverviewUrl(screenshots, page.id, issue.scanner ? [issue.scanner, 'axe'] : ['axe'])
			: null
	);
	const overviewViewBox = $derived.by(() => {
		if (!overview) return null;
		if (overviewEl) {
			return getCroppedViewBox(overview.pageWidth, overview.pageHeight, overviewEl, {
				minWidth: 360,
				minHeight: 220,
				padding: 64
			});
		}
		return {
			x: 0,
			y: 0,
			width: Math.min(overview.pageWidth, 360),
			height: Math.min(overview.pageHeight, 220)
		};
	});
	const screenshotUrl = $derived(
		showScreenshot
			? getIssueScreenshotUrl({
					screenshots,
					scannerId: issue.scanner,
					issueId: issue.id,
					pageId: issue.pageId
				})
			: null
	);
</script>

<button
	type="button"
	{onclick}
	class={cn(
		'hover:bg-surface-muted/70 focus-visible:ring-accent/35 flex w-full items-start gap-3 text-left transition-colors focus-visible:ring-4 focus-visible:outline-none',
		showScreenshot ? 'min-h-[120px] p-4 sm:px-5' : 'px-4 py-3',
		isVirtualized && showScreenshot && 'h-[120px] overflow-hidden',
		isSelected && 'bg-accent/5 ring-1 ring-accent/20'
	)}
	data-testid="issue-row"
>
	<span
		class={cn(
			'mt-0.5 shrink-0 rounded-md px-2 py-1 text-[10px] leading-none font-semibold tracking-wide uppercase',
			getSeverityBadgeClass(issue.severity)
		)}
	>
		{issue.severity}
	</span>
	<div class="min-w-0 flex-1">
		<p
			class={cn(
				'text-ink text-base leading-snug font-semibold',
				isVirtualized && showScreenshot ? 'line-clamp-1' : 'line-clamp-2'
			)}
		>
			{issue.title}
		</p>
		<p class="text-ink-muted mt-1.5 line-clamp-2 text-sm leading-relaxed">
			{issue.description}
		</p>
		{#if showScreenshot}
			<div class="mt-2.5 flex flex-wrap items-center gap-2 text-xs">
				<span class="text-ink-faint">
					{issue.scanner} &middot; {issue.elementCount} element{issue.elementCount !== 1
						? 's'
						: ''}
				</span>
				{#if issue.wcagTags?.length}
					{#each issue.wcagTags.slice(0, 3) as tag (tag)}
						<span class="bg-surface-muted text-ink-muted rounded-md px-1.5 py-0.5">
							{tag}
						</span>
					{/each}
					{#if issue.wcagTags.length > 3}
						<span class="text-ink-faint">+{issue.wcagTags.length - 3} more</span>
					{/if}
				{/if}
			</div>
		{/if}
	</div>
	{#if showScreenshot}
		{#if overviewUrl && overview && overviewViewBox}
			<div class="bg-surface border-line h-16 w-24 shrink-0 overflow-hidden rounded-md border shadow-xs">
				<svg
					class="bg-surface-muted h-full w-full"
					viewBox={`${overviewViewBox.x} ${overviewViewBox.y} ${overviewViewBox.width} ${overviewViewBox.height}`}
					preserveAspectRatio="xMidYMid slice"
					role="img"
					aria-label="Page screenshot crop"
				>
					<image
						href={overviewUrl}
						x="0"
						y="0"
						width={overview.pageWidth}
						height={overview.pageHeight}
					/>
					{#if overviewEl}
						<rect
							x={overviewEl.x}
							y={overviewEl.y}
							width={overviewEl.width}
							height={overviewEl.height}
							fill="transparent"
							stroke={getSeverityStrokeColor(issue.severity)}
							stroke-width="8"
						/>
					{/if}
				</svg>
			</div>
		{:else if screenshotUrl}
			<div class="border-line h-16 w-24 shrink-0 overflow-hidden rounded-md border shadow-xs">
				<img
					src={screenshotUrl}
					alt="Issue screenshot"
					class="h-full w-full object-cover"
					loading="lazy"
				/>
			</div>
		{/if}
	{/if}
</button>
