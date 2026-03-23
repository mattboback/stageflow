<script lang="ts">
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, UnifiedReport } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';
	import { getPageOverviewUrl, getSeverityBadgeClass } from '$lib/report';
	import { cn } from '$lib/utils';

	import PageOverviewViewer from './PageOverviewViewer.svelte';

	interface Props {
		report: UnifiedReport;
		screenshots: ScreenshotArtifact[];
		activeScanner: string | null;
		activePage: string | null;
		onSelectPage: (pageId: string) => void;
		onIssueSelect: (issue: IssueDetail, highlightedElementId?: string) => void;
	}

	let { report, screenshots, activeScanner, activePage, onSelectPage, onIssueSelect }: Props =
		$props();

	const selectedPage = $derived(
		report.pages.find((page) => page.id === activePage) ?? report.pages[0] ?? null
	);

	const issuesByPage = $derived.by(() => {
		const map: Record<string, IssueDetail[]> = {};
		for (const issue of report.issues) {
			if (activeScanner && issue.scanner !== activeScanner) {
				continue;
			}
			const list = map[issue.pageId] ?? [];
			list.push(issue);
			map[issue.pageId] = list;
		}
		return map;
	});

	const selectedIssues = $derived(selectedPage ? (issuesByPage[selectedPage.id] ?? []) : []);

	const overviewUrl = $derived(
		selectedPage
			? getPageOverviewUrl(
					screenshots,
					selectedPage.id,
					activeScanner ? [activeScanner, 'axe'] : ['axe']
				)
			: null
	);
</script>

{#snippet pageRow(page: UnifiedReport['pages'][number])}
	{@const pageIssues = issuesByPage[page.id] ?? []}
	<button
		onclick={() => onSelectPage(page.id)}
		class={cn(
			'border-line flex w-full items-center justify-between border-b px-4 py-3 text-left text-sm transition',
			selectedPage?.id === page.id ? 'bg-accent/5 text-accent' : 'hover:bg-surface-muted'
		)}
	>
		<div class="min-w-0">
			<p class="text-ink truncate font-medium">{page.path ?? page.url}</p>
			<p class="text-ink-muted text-xs">{pageIssues.length} issues</p>
		</div>
		<span class="text-ink-faint text-xs">{(page.durationMs / 1000).toFixed(1)}s</span>
	</button>
{/snippet}

{#snippet issueRow(issue: IssueDetail)}
	<button
		onclick={() => onIssueSelect(issue)}
		class="hover:bg-surface-muted flex w-full items-start gap-3 px-4 py-3 text-left text-sm transition"
	>
		<span
			class={cn(
				'mt-0.5 rounded px-2 py-0.5 text-xs font-semibold uppercase',
				getSeverityBadgeClass(issue.severity)
			)}
		>
			{issue.severity}
		</span>
		<div class="min-w-0">
			<p class="text-ink font-medium">{issue.title}</p>
			<p class="text-ink-muted mt-1 line-clamp-2 text-xs">{issue.description}</p>
		</div>
	</button>
{/snippet}

<div class="grid grid-cols-1 gap-6 lg:grid-cols-[280px,1fr]">
	<Panel class="max-h-[720px] overflow-hidden shadow-sm" padding="none" rounded="2xl">
		<div class="border-line border-b p-4">
			<h3 class="text-ink text-base leading-none font-semibold tracking-tight">Pages</h3>
		</div>
		<div class="max-h-[640px] overflow-y-auto">
			{#each report.pages as page (page.id)}
				{@render pageRow(page)}
			{/each}
		</div>
	</Panel>

	{#if selectedPage}
		<div class="space-y-4">
			<Panel class="shadow-sm" padding="none" rounded="2xl">
				<div class="border-line border-b p-4">
					<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
						{selectedPage.path ?? selectedPage.url}
					</h3>
					<p class="text-ink-muted mt-1 text-sm">{selectedIssues.length} issues detected</p>
				</div>
				<div class="p-4">
					<PageOverviewViewer
						page={selectedPage}
						issues={selectedIssues}
						screenshotUrl={overviewUrl}
						onSelectIssue={onIssueSelect}
					/>
				</div>
			</Panel>

			<Panel class="shadow-sm" padding="none" rounded="2xl">
				<div class="border-line border-b p-4">
					<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
						Issues on this page
					</h3>
				</div>
				{#if selectedIssues.length === 0}
					<div class="text-ink-muted p-6 text-center text-sm">No issues for this page.</div>
				{:else}
					<div class="divide-line divide-y">
						{#each selectedIssues as issue (issue.id)}
							{@render issueRow(issue)}
						{/each}
					</div>
				{/if}
			</Panel>
		</div>
	{:else}
		<Panel class="shadow-sm" padding="sm" rounded="2xl">
			<div class="text-ink-muted text-center text-sm">No pages available.</div>
		</Panel>
	{/if}
</div>
