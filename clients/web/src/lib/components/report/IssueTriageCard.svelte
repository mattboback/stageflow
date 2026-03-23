<script lang="ts">
	import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';
	import { getCroppedViewBox, getSeverityBadgeClass } from '$lib/report';
	import { cn } from '$lib/utils';

	interface Props {
		issue: IssueDetail;
		page: PageSummary | null;
		pageOverviewUrl: string | null;
		suggestedOwner: string;
		primaryFix: string | null;
	}

	let { issue, page, pageOverviewUrl, suggestedOwner, primaryFix }: Props = $props();

	const triageOverviewCrop = $derived.by(() => {
		if (!page?.pageOverview) return null;
		const overview = page.pageOverview;
		const elements = overview.elements ?? [];
		const element = elements.find((el) => el.issueId === issue.id) ?? null;
		if (!element) return null;

		const viewBox = getCroppedViewBox(overview.pageWidth, overview.pageHeight, element);
		if (!viewBox) return null;

		return { overview, element, viewBox };
	});
</script>

<Panel padding="lg" rounded="2xl" class="bg-surface-muted">
	<div class="flex flex-wrap items-start justify-between gap-4">
		<div class="min-w-0">
			<p class="text-ink-muted text-xs font-semibold tracking-wide uppercase">Triage</p>
			<div class="mt-2 flex flex-wrap items-center gap-2">
				<span
					class={cn(
						'inline-block rounded px-2 py-0.5 text-xs font-semibold uppercase',
						getSeverityBadgeClass(issue.severity)
					)}
				>
					{issue.severity}
				</span>
				<span
					class="border-line text-ink-muted rounded-full border px-2 py-0.5 text-xs font-semibold"
				>
					Suggested owner: {suggestedOwner}
				</span>
			</div>

			{#if primaryFix}
				<p class="text-ink mt-3 text-sm font-semibold">Fix (1-line)</p>
				<p class="text-ink-muted mt-1 text-sm leading-relaxed whitespace-pre-wrap">
					{primaryFix}
				</p>
			{/if}

			{#if issue.userImpact?.statement}
				<p class="text-ink mt-4 text-sm font-semibold">Impact</p>
				<p class="text-ink-muted mt-1 text-sm leading-relaxed">{issue.userImpact.statement}</p>
			{/if}

			<p class="text-ink-faint mt-4 text-xs">
				Page: {page?.path ?? page?.url ?? issue.pageUrl}
			</p>
		</div>

		<div class="w-full max-w-[440px]">
			{#if triageOverviewCrop && pageOverviewUrl}
				<div>
					<p class="text-ink-muted mb-2 text-xs font-semibold tracking-wide uppercase">Evidence</p>
					<Panel padding="none" rounded="lg" class="overflow-hidden bg-white">
						<svg
							class="h-[220px] w-full bg-slate-50"
							viewBox={`${triageOverviewCrop.viewBox.x} ${triageOverviewCrop.viewBox.y} ${triageOverviewCrop.viewBox.width} ${triageOverviewCrop.viewBox.height}`}
							preserveAspectRatio="xMidYMid slice"
							role="img"
							aria-label="Page screenshot crop"
						>
							<image
								href={pageOverviewUrl}
								x="0"
								y="0"
								width={triageOverviewCrop.overview.pageWidth}
								height={triageOverviewCrop.overview.pageHeight}
							/>
							<rect
								x={triageOverviewCrop.element.x}
								y={triageOverviewCrop.element.y}
								width={triageOverviewCrop.element.width}
								height={triageOverviewCrop.element.height}
								fill="rgba(0,0,0,0)"
								stroke="rgba(15,118,110,0.95)"
								stroke-width="6"
							/>
						</svg>
					</Panel>
				</div>
			{/if}
		</div>
	</div>
</Panel>
