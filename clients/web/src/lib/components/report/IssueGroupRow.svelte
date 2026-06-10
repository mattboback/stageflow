<script lang="ts">
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, IssueGroup, PageSummary } from '$lib/types/unified-report';

	import { StatusPill, type StatusPillTone } from '$lib/components/ui';
	import { rewriteIssueTitle } from '$lib/report';
	import { cn } from '$lib/utils';
	import { ChevronDown, ChevronRight } from 'lucide-svelte';

	import IssueRowCard from './IssueRowCard.svelte';

	interface Props {
		group: IssueGroup;
		pagesById: Record<string, PageSummary>;
		screenshots: ScreenshotArtifact[];
		showPreviews?: boolean;
		expanded?: boolean;
		selectedIssueId?: string | null;
		onToggle: () => void;
		onIssueSelect: (issue: IssueDetail) => void;
	}

	let {
		group,
		pagesById,
		screenshots,
		showPreviews = false,
		expanded = false,
		selectedIssueId = null,
		onToggle,
		onIssueSelect
	}: Props = $props();

	const severityToTone: Record<string, StatusPillTone> = {
		critical: 'failing',
		serious: 'high-risk',
		moderate: 'needs-work',
		minor: 'watch',
		info: 'neutral'
	};

	const tone = $derived<StatusPillTone>(severityToTone[group.severity] ?? 'neutral');
	const occurrenceCount = $derived(group.occurrences.length);
	const pageCount = $derived(group.pageIds.length);
	// Manual-audit titles ship as positive statements ("controls are focusable")
	// that read like a pass — rewrite them to "Verify: …" like the modal does.
	const displayTitle = $derived(
		group.occurrences[0] ? rewriteIssueTitle(group.occurrences[0]) : group.title
	);
</script>

<div class="border-line/70 bg-surface border-b last:border-b-0" data-testid="issue-group-row">
	<button
		type="button"
		onclick={onToggle}
		aria-expanded={expanded}
		class={cn(
			'hover:bg-surface-muted/60 flex w-full items-start gap-3 px-4 py-3 text-left transition sm:px-5',
			expanded && 'bg-surface-muted/40'
		)}
	>
		<span class="text-ink-faint mt-1 flex h-4 w-4 shrink-0 items-center justify-center">
			{#if expanded}
				<ChevronDown class="h-4 w-4" />
			{:else}
				<ChevronRight class="h-4 w-4" />
			{/if}
		</span>
		<div class="min-w-0 flex-1">
			<div class="flex items-start justify-between gap-3">
				<div class="min-w-0 flex-1">
					<div class="flex items-center gap-2">
						<StatusPill {tone} label={group.severity} size="sm" />
						<span class="text-ink-faint font-mono text-[11px] tracking-wide uppercase">
							{group.ruleId}
						</span>
					</div>
					<h4 class="text-ink mt-1.5 truncate text-sm leading-snug font-semibold">
						{displayTitle}
					</h4>
					<div class="mt-1.5 flex flex-wrap items-center gap-2 text-xs">
						<span
							class={cn(
								'inline-flex items-center rounded-full border px-2 py-0.5 font-mono text-[11px] leading-none font-bold tabular-nums',
								occurrenceCount > 1
									? 'border-line bg-surface-muted text-ink'
									: 'border-line/60 bg-surface text-ink-muted'
							)}
						>
							×{occurrenceCount.toLocaleString()}
						</span>
						<span class="text-ink-muted">
							{occurrenceCount === 1 ? 'occurrence' : 'occurrences'} ·
							<span class="text-ink font-mono font-semibold">{pageCount.toLocaleString()}</span>
							{pageCount === 1 ? 'page' : 'pages'}
						</span>
						{#if occurrenceCount >= 5}
							<span
								class="inline-flex items-center rounded-full border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-[11px] leading-none font-semibold text-emerald-700"
							>
								Fix once → closes {occurrenceCount.toLocaleString()} instances
							</span>
						{/if}
					</div>
				</div>
				<span
					class="text-ink-faint shrink-0 font-mono text-[11px] tracking-wide uppercase"
					data-testid="group-scanner"
				>
					{group.scanner}
				</span>
			</div>
		</div>
	</button>

	{#if expanded}
		<div class="border-line/60 divide-line/70 divide-y border-t" data-testid="group-occurrences">
			{#each group.occurrences as issue (issue.id)}
				<IssueRowCard
					{issue}
					page={pagesById[issue.pageId] ?? null}
					{screenshots}
					showScreenshot={showPreviews}
					isSelected={selectedIssueId === issue.id}
					inGroup={true}
					onclick={() => onIssueSelect(issue)}
				/>
			{/each}
		</div>
	{/if}
</div>
