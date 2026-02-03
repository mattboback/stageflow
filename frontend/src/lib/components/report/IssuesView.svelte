<script lang="ts">
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, UnifiedReport } from '$lib/types/unified-report';

	import { Input, Panel, Select, Tabs } from '$lib/components/ui';
	import {
		filterIssues,
		getSeverityChipClass,
		isIssueSortKey,
		ISSUE_SORTS,
		ISSUE_SORT_LABELS,
		sortIssues,
		getVirtualWindow
	} from '$lib/report';
	import { cn } from '$lib/utils';
	import { Search } from 'lucide-svelte';

	import IssueRowCard from './IssueRowCard.svelte';

	interface Props {
		report: UnifiedReport;
		screenshots: ScreenshotArtifact[];
		activeScanner: string | null;
		activePage: string | null;
		activeSeverity: string | null;
		activeCategory: string | null;
		searchTerm: string;
		issueSort: string;
		selectedIssueId: string | null;
		onScannerChange: (scannerId: string | null) => void;
		onPageChange: (pageId: string | null) => void;
		onSeverityChange: (severity: string | null) => void;
		onCategoryChange: (category: string | null) => void;
		onSearchChange: (query: string) => void;
		onSortChange: (sort: string) => void;
		onIssueSelect: (issue: IssueDetail) => void;
		onClearFilters: () => void;
	}

	const {
		report,
		screenshots,
		activeScanner,
		activePage,
		activeSeverity,
		activeCategory,
		searchTerm,
		issueSort,
		selectedIssueId,
		onScannerChange,
		onPageChange,
		onSeverityChange,
		onCategoryChange,
		onSearchChange,
		onSortChange,
		onIssueSelect,
		onClearFilters
	}: Props = $props();

	const severityOptions = ['all', 'critical', 'serious', 'moderate', 'minor', 'info'] as const;

	const scannerTabs = $derived.by(() => {
		const byScanner = report.summary.byScanner ?? {};
		return [
			{ id: 'all', label: `All (${report.issues.length.toLocaleString()})` },
			...report.scanners.map((scanner) => ({
				id: scanner.id,
				label: `${scanner.name ?? scanner.id}${byScanner[scanner.id] ? ` (${byScanner[scanner.id].toLocaleString()})` : ''}`
			}))
		];
	});

	const categories = $derived.by(() => {
		const seen: Record<string, true> = {};
		for (const issue of report.issues) {
			if (issue.category) seen[issue.category] = true;
		}
		return Object.keys(seen).sort();
	});

	const pagesById = $derived(Object.fromEntries(report.pages.map((page) => [page.id, page])));
	const filteredIssues = $derived(
		filterIssues(report.issues, {
			scannerId: activeScanner,
			pageId: activePage,
			severity: activeSeverity,
			category: activeCategory,
			query: searchTerm
		})
	);

	const sortedIssues = $derived(
		sortIssues(filteredIssues, isIssueSortKey(issueSort) ? issueSort : 'severity')
	);

	const hasActiveFilters = $derived(
		Boolean(activeScanner || activePage || activeSeverity || activeCategory || searchTerm.trim())
	);

	// Debounced search: local state syncs immediately for responsive UI,
	// but URL update is debounced to avoid expensive recalculations on each keystroke.
	// We use $state here instead of writable $derived because:
	// 1. We need to sync from prop (searchTerm) when it changes externally
	// 2. We need to allow local mutations from user input
	// 3. We need to debounce before calling the callback
	const SEARCH_DEBOUNCE_MS = 250;
	// eslint-disable-next-line svelte/prefer-writable-derived -- complex bidirectional sync with debounce
	let localSearchValue = $state('');

	// Sync local value when searchTerm prop changes externally (e.g., clear filters, initial load)
	$effect(() => {
		localSearchValue = searchTerm;
	});

	// Debounce the URL update when local value changes
	$effect(() => {
		// Capture the current value for the closure
		const currentValue = localSearchValue;
		// Read searchTerm inside the effect so it's tracked as a dependency
		const propValue = searchTerm;

		// Skip if already in sync with prop
		if (currentValue === propValue) return;

		const timer = setTimeout(() => {
			onSearchChange(currentValue.trim());
		}, SEARCH_DEBOUNCE_MS);

		// Cleanup: cancel pending timeout if effect re-runs or component unmounts
		return () => clearTimeout(timer);
	});

	function handleSearchInput(value: string) {
		localSearchValue = value;
	}

	let listContainer = $state<HTMLDivElement | null>(null);
	let scrollTop = $state(0);
	let viewportHeight = $state(600);
	const rowHeight = 120;
	const overscan = 6;

	const shouldVirtualize = $derived(sortedIssues.length > 200);
	const totalHeight = $derived(sortedIssues.length * rowHeight);
	const virtualWindow = $derived.by(() =>
		getVirtualWindow({
			scrollTop,
			viewportHeight,
			rowHeight,
			totalItems: sortedIssues.length,
			overScan: overscan
		})
	);
	const visibleIssues = $derived(
		shouldVirtualize
			? sortedIssues.slice(virtualWindow.startIndex, virtualWindow.endIndex)
			: sortedIssues
	);
	const offsetY = $derived(shouldVirtualize ? virtualWindow.offset : 0);

</script>



<div class="space-y-4">
	<Tabs
		tabs={scannerTabs}
		value={activeScanner ?? 'all'}
		onValueChange={(id) => onScannerChange(id === 'all' ? null : id)}
	/>

	<Panel class="shadow-sm" padding="none" rounded="2xl">
		<div class="border-line border-b p-4">
			<div class="flex flex-col gap-3">
				<div class="flex items-center justify-between">
					<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
						Issues ({filteredIssues.length}{filteredIssues.length !== report.issues.length
							? ` / ${report.issues.length}`
							: ''})
					</h3>
					{#if hasActiveFilters}
						<button onclick={onClearFilters} class="text-accent text-sm hover:underline">
							Clear filters
						</button>
					{/if}
				</div>
				<div class="grid gap-3 lg:grid-cols-[1.4fr,1fr]">
					<div class="relative">
						<Search class="text-ink-faint absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
						<Input
							value={localSearchValue}
							placeholder="Search issues"
							class="pl-9"
							oninput={(event) =>
								handleSearchInput((event.currentTarget as HTMLInputElement).value)}
						/>
					</div>
					<div class="flex flex-wrap items-center justify-end gap-3">
						<label for="issue-sort" class="text-ink-muted text-xs">Sort</label>
						<Select
							id="issue-sort"
							uiSize="sm"
							value={issueSort}
							onchange={(event) => onSortChange((event.currentTarget as HTMLSelectElement).value)}
						>
							{#each ISSUE_SORTS as sort (sort)}
								<option value={sort}>{ISSUE_SORT_LABELS[sort]}</option>
							{/each}
						</Select>
					</div>
				</div>
				<div class="flex flex-wrap items-center gap-2">
					{#each severityOptions as severity (severity)}
						<button
							onclick={() => onSeverityChange(severity === 'all' ? null : severity)}
							class={getSeverityChipClass(
								severity,
								activeSeverity === null ? severity === 'all' : activeSeverity === severity
							)}
						>
							{severity}
						</button>
					{/each}
				</div>
				<div class="grid gap-3 sm:grid-cols-2">
					<label class="text-ink-muted text-xs">
						Category
						<Select
							class="mt-1"
							value={activeCategory ?? 'all'}
							onchange={(event) => {
								const value = (event.currentTarget as HTMLSelectElement).value;
								onCategoryChange(value === 'all' ? null : value);
							}}
						>
							<option value="all">All categories</option>
							{#each categories as category (category)}
								<option value={category}>{category}</option>
							{/each}
						</Select>
					</label>
					<label class="text-ink-muted text-xs">
						Page
						<Select
							class="mt-1"
							value={activePage ?? 'all'}
							onchange={(event) => {
								const value = (event.currentTarget as HTMLSelectElement).value;
								onPageChange(value === 'all' ? null : value);
							}}
						>
							<option value="all">All pages</option>
							{#each report.pages as page (page.id)}
								<option value={page.id}>{page.path ?? page.url}</option>
							{/each}
						</Select>
					</label>
				</div>
			</div>
		</div>

		{#if filteredIssues.length === 0}
			<div class="flex flex-col items-center justify-center py-12 text-center">
				<p class="text-ink font-medium">No issues found</p>
				<p class="text-ink-muted mt-1 text-sm">
					{#if hasActiveFilters}
						Try adjusting your filters
					{:else}
						This scan completed with no issues
					{/if}
				</p>
			</div>
		{:else}
			<div
				bind:this={listContainer}
				class={cn('divide-line divide-y', shouldVirtualize && 'max-h-[720px] overflow-y-auto')}
				data-testid="issue-list"
				onscroll={() => {
					if (!listContainer) return;
					scrollTop = listContainer.scrollTop;
					viewportHeight = listContainer.clientHeight;
				}}
			>
				{#if shouldVirtualize}
					<div style={`height: ${totalHeight}px; position: relative;`}>
						<div style={`transform: translateY(${offsetY}px);`}>
							{#each visibleIssues as issue (issue.id)}
								<IssueRowCard
									{issue}
									page={pagesById[issue.pageId] ?? null}
									{screenshots}
									showScreenshot={true}
									isVirtualized={true}
									isSelected={selectedIssueId === issue.id}
									onclick={() => onIssueSelect(issue)}
								/>
							{/each}
						</div>
					</div>
				{:else}
					{#each sortedIssues as issue (issue.id)}
						<IssueRowCard
							{issue}
							page={pagesById[issue.pageId] ?? null}
							{screenshots}
							showScreenshot={true}
							isVirtualized={false}
							isSelected={selectedIssueId === issue.id}
							onclick={() => onIssueSelect(issue)}
						/>
					{/each}
				{/if}
			</div>
		{/if}
	</Panel>
</div>
