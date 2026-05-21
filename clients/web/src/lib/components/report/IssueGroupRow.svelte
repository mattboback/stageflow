<script lang="ts">
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, IssueGroup, PageSummary } from '$lib/types/unified-report';

	import { StatusPill, type StatusPillTone } from '$lib/components/ui';
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
						{group.title}
					</h4>
					<div class="text-ink-muted mt-1 flex items-center gap-3 text-xs">
						<span>
							Found on
							<span class="text-ink font-mono font-semibold"
								>{occurrenceCount.toLocaleString()}</span
							>
							{occurrenceCount === 1 ? 'element' : 'elements'} across
							<span class="text-ink font-mono font-semibold">{pageCount.toLocaleString()}</span>
							{pageCount === 1 ? 'page' : 'pages'}
						</span>
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
