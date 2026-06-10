<script lang="ts">
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

	import {
		getCroppedViewBox,
		getIssueScreenshotUrl,
		getPageOverviewUrl,
		getSeverityContainerClass,
		getSeverityDotClass,
		getSeverityStrokeColor,
		rewriteIssueTitle
	} from '$lib/report';
	import { cn } from '$lib/utils';

	interface Props {
		issue: IssueDetail;
		page: PageSummary | null;
		screenshots: ScreenshotArtifact[];
		showScreenshot?: boolean;
		isVirtualized?: boolean;
		isSelected?: boolean;
		/**
		 * When true, the row is rendered inside an expanded rule group — show
		 * occurrence-specific context (page URL + selector + element label)
		 * instead of repeating the rule title and description.
		 */
		inGroup?: boolean;
		onclick?: () => void;
	}

	let {
		issue,
		page = null,
		screenshots,
		showScreenshot = true,
		isVirtualized = false,
		isSelected = false,
		inGroup = false,
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

	const primaryOccurrence = $derived(issue.occurrences?.[0] ?? null);
	const pageLabel = $derived(page?.path ?? page?.url ?? null);
	const displayTitle = $derived(rewriteIssueTitle(issue));

	// In group context, the "primary line" is the page+selector context so
	// occurrences don't all look identical. Otherwise it's the rule title.
	const primaryLine = $derived.by(() => {
		if (!inGroup) return displayTitle;
		const selector = primaryOccurrence?.selector ?? primaryOccurrence?.label ?? null;
		if (pageLabel && selector) return `${pageLabel} · ${selector}`;
		if (selector) return selector;
		if (pageLabel) return pageLabel;
		return displayTitle;
	});

	const secondaryLine = $derived.by(() => {
		// Prefer the plain-English element label over the rule's technical
		// description — "Image in the page header" beats spec prose at a glance.
		if (!inGroup) return issue.friendlyNode?.label ?? issue.description;
		const summary = primaryOccurrence?.failureSummary ?? primaryOccurrence?.textSnippet ?? null;
		if (summary) return summary;
		return issue.description;
	});

	const affectedGroups = $derived(issue.userImpact?.affectedGroups ?? []);
	const affectedLabel = $derived(
		affectedGroups.includes('all') ? 'all users' : affectedGroups.join(', ')
	);
</script>

<button
	type="button"
	{onclick}
	class={cn(
		'hover:bg-surface-muted/70 focus-visible:ring-accent/35 flex w-full items-start gap-3 text-left transition-colors focus-visible:ring-4 focus-visible:outline-none',
		showScreenshot ? 'min-h-[120px] p-4 sm:px-5' : 'px-4 py-3',
		isVirtualized && showScreenshot && 'h-[120px] overflow-hidden',
		isSelected && 'bg-accent/5 ring-accent/20 ring-1'
	)}
	data-testid="issue-row"
>
	<span
		class={cn(
			'mt-0.5 inline-flex shrink-0 items-center gap-1.5 rounded-md border px-2 py-1 text-[10px] leading-none font-semibold tracking-wide uppercase',
			getSeverityContainerClass(issue.severity)
		)}
	>
		<span
			class={cn(
				'h-2 w-2 rounded-full',
				getSeverityDotClass(issue.severity),
				issue.severity === 'critical' && 'animate-hotspot-pulse text-red-500'
			)}
		></span>
		{issue.severity}
	</span>
	<div class="min-w-0 flex-1">
		<p
			class={cn(
				'text-ink text-base leading-snug font-semibold',
				inGroup && 'font-mono text-sm break-all',
				isVirtualized && showScreenshot ? 'line-clamp-1' : 'line-clamp-2'
			)}
		>
			{primaryLine}
		</p>
		{#if secondaryLine}
			<p class="text-ink-muted mt-1.5 line-clamp-2 text-sm leading-relaxed">
				{secondaryLine}
			</p>
		{/if}
		{#if showScreenshot}
			<div
				class="text-ink-faint mt-2.5 flex min-w-0 flex-wrap items-center gap-x-2 gap-y-0.5 font-mono text-[10px]"
			>
				<span>
					{issue.scanner} &middot; {issue.elementCount} element{issue.elementCount !== 1 ? 's' : ''}
				</span>
				{#if !inGroup && issue.wcagTags?.length}
					<span class="text-ink-faint/80">
						&middot; {issue.wcagTags.slice(0, 3).join(', ')}{issue.wcagTags.length > 3
							? ` +${issue.wcagTags.length - 3}`
							: ''}
					</span>
				{:else if inGroup && pageLabel}
					<span class="truncate">&middot; {pageLabel}</span>
				{/if}
				{#if affectedGroups.length}
					<span class="font-sans">&middot; Affects: {affectedLabel}</span>
				{/if}
			</div>
		{/if}
	</div>
	{#if showScreenshot}
		{#if overviewUrl && overview && overviewViewBox}
			<div
				class="bg-surface border-line h-16 w-24 shrink-0 overflow-hidden rounded-md border shadow-xs"
			>
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
					decoding="async"
				/>
			</div>
		{/if}
	{/if}
</button>
