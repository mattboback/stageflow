<script lang="ts">
	import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';
	import { getSeverityStrokeColor } from '$lib/report';

	interface Props {
		issue: IssueDetail;
		page: PageSummary | null;
		screenshotUrl: string | null;
		pageOverviewUrl: string | null;
		showPageOverview?: boolean;
		onElementClick?: (elementId: string) => void;
	}

	const {
		issue,
		page,
		screenshotUrl,
		pageOverviewUrl,
		showPageOverview = true,
		onElementClick
	}: Props = $props();

	const pageOverviewElements = $derived.by(() => {
		const elements = page?.pageOverview?.elements ?? [];
		return elements.filter((el) => el.issueId === issue.id);
	});

	const pageWidth = $derived(page?.pageOverview?.pageWidth ?? 0);
	const pageHeight = $derived(page?.pageOverview?.pageHeight ?? 0);
	const strokeColor = $derived(getSeverityStrokeColor(issue.severity));
</script>

<div class="space-y-4">
	<h3 class="text-ink font-semibold">Evidence</h3>
	{#if screenshotUrl}
		<div>
			<p class="text-ink-muted mb-2 text-sm">Scanner screenshot</p>
			<Panel padding="none" rounded="lg" class="overflow-hidden">
				<img src={screenshotUrl} alt="Issue highlighted on page" class="w-full" loading="lazy" />
			</Panel>
		</div>
	{/if}

	{#if showPageOverview && page && pageOverviewUrl && pageWidth > 0 && pageHeight > 0}
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
						<image
							href={pageOverviewUrl}
							x="0"
							y="0"
							width={pageWidth}
							height={pageHeight}
						/>
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
			{#if pageOverviewElements.length}
				<p class="text-ink-muted mt-2 text-xs">
					Click a highlight box to jump to the matching element details below.
				</p>
			{/if}
		</div>
	{/if}
</div>
