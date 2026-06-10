<script lang="ts">
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, IssueGroup, UnifiedReport } from '$lib/types/unified-report';

	import { Panel, Select } from '$lib/components/ui';
	import {
		ISSUE_SORTS,
		ISSUE_SORT_LABELS,
		getVirtualWindow,
		groupIssuesByRule,
		isIssueSortKey,
		isManualReviewIssue,
		sortIssues
	} from '$lib/report';
	import { cn } from '$lib/utils';
	import { ChevronDown, ChevronRight, Eye, Filter, X } from 'lucide-svelte';

	import IssueFilterSidebar from './IssueFilterSidebar.svelte';
	import IssueGroupRow from './IssueGroupRow.svelte';
	import IssueRowCard from './IssueRowCard.svelte';

	interface Props {
		report: UnifiedReport;
		screenshots: ScreenshotArtifact[];
		activeScanners: string[];
		activePage: string | null;
		activeSeverities: string[];
		activeCategory: string | null;
		searchTerm: string;
		issueSort: string;
		selectedIssueId: string | null;
		onScannersChange: (values: string[]) => void;
		onPageChange: (pageId: string | null) => void;
		onSeveritiesChange: (values: string[]) => void;
		onCategoryChange: (category: string | null) => void;
		onSearchChange: (query: string) => void;
		onSortChange: (sort: string) => void;
		onIssueSelect: (issue: IssueDetail) => void;
		onClearFilters: () => void;
	}

	let {
		report,
		screenshots,
		activeScanners,
		activePage,
		activeSeverities,
		activeCategory,
		searchTerm,
		issueSort,
		selectedIssueId,
		onScannersChange,
		onPageChange,
		onSeveritiesChange,
		onCategoryChange,
		onSearchChange,
		onSortChange,
		onIssueSelect,
		onClearFilters
	}: Props = $props();

	const pagesById = $derived(Object.fromEntries(report.pages.map((page) => [page.id, page])));
	const activePageLabel = $derived(
		activePage ? (pagesById[activePage]?.path ?? pagesById[activePage]?.url ?? activePage) : null
	);

	const filteredIssues = $derived.by(() => {
		const q = searchTerm.trim().toLowerCase();
		return report.issues.filter((issue) => {
			if (activeScanners.length && !activeScanners.includes(issue.scanner)) return false;
			if (activePage && issue.pageId !== activePage) return false;
			if (activeSeverities.length && !activeSeverities.includes(issue.severity)) return false;
			if (activeCategory && issue.category !== activeCategory) return false;
			if (q) {
				const haystack = `${issue.title} ${issue.description} ${issue.ruleId}`.toLowerCase();
				if (!haystack.includes(q)) return false;
			}
			return true;
		});
	});

	const sortedIssues = $derived(
		sortIssues(filteredIssues, isIssueSortKey(issueSort) ? issueSort : 'severity')
	);

	const groups = $derived<IssueGroup[]>(groupIssuesByRule(filteredIssues));

	// Manual-review rules (e.g. "verify color contrast" checks) are noise next to
	// concrete failures — park them in a collapsed bucket below the actionable list.
	const actionableGroups = $derived(
		groups.filter((group) => !group.occurrences.every(isManualReviewIssue))
	);
	const manualGroups = $derived(
		groups.filter((group) => group.occurrences.every(isManualReviewIssue))
	);
	const manualCheckCount = $derived(
		manualGroups.reduce((sum, group) => sum + group.occurrences.length, 0)
	);

	let manualOpen = $state(false);
	// If everything needs manual review, hiding the whole list behind a collapse
	// would render an empty panel — force the bucket open in that case.
	const manualExpanded = $derived(manualOpen || actionableGroups.length === 0);

	const hasActiveFilters = $derived(
		activeScanners.length > 0 ||
			activeSeverities.length > 0 ||
			activePage !== null ||
			activeCategory !== null ||
			searchTerm.trim().length > 0
	);

	const activeFilterCount = $derived(
		activeScanners.length +
			activeSeverities.length +
			(activePage ? 1 : 0) +
			(activeCategory ? 1 : 0) +
			(searchTerm.trim() ? 1 : 0)
	);

	let viewMode = $state<'grouped' | 'flat'>('grouped');
	let previewMode = $state<'auto' | 'on' | 'off'>('auto');
	let isCompactViewport = $state(false);
	let mobileFiltersOpen = $state(false);
	let expandedGroups = $state<Record<string, boolean>>({});

	const showPreviews = $derived.by(() => {
		if (previewMode === 'on') return true;
		if (previewMode === 'off') return false;
		return !isCompactViewport;
	});

	let listContainer = $state<HTMLDivElement | null>(null);
	let scrollTop = $state(0);
	let viewportHeight = $state(600);
	let scrollRaf = $state<number | null>(null);

	const rowHeight = $derived(showPreviews ? 120 : 88);
	const overscan = $derived(isCompactViewport ? 4 : 6);

	const shouldVirtualize = $derived(viewMode === 'flat' && sortedIssues.length > 200);
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

	function handleListScroll() {
		if (!listContainer) return;
		if (scrollRaf !== null) return;
		scrollRaf = requestAnimationFrame(() => {
			scrollRaf = null;
			if (!listContainer) return;
			scrollTop = listContainer.scrollTop;
			viewportHeight = listContainer.clientHeight;
		});
	}

	$effect(() => {
		if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
			isCompactViewport = false;
			return;
		}
		const media = window.matchMedia('(max-width: 1023px)');
		const update = (event?: MediaQueryListEvent) => {
			isCompactViewport = event ? event.matches : media.matches;
		};
		update();
		media.addEventListener('change', update);
		return () => media.removeEventListener('change', update);
	});

	$effect(() => {
		if (!listContainer || typeof ResizeObserver === 'undefined') return;
		const observer = new ResizeObserver(() => {
			if (!listContainer) return;
			viewportHeight = listContainer.clientHeight;
		});
		observer.observe(listContainer);
		viewportHeight = listContainer.clientHeight;
		return () => observer.disconnect();
	});

	$effect(() => {
		return () => {
			if (scrollRaf !== null) cancelAnimationFrame(scrollRaf);
		};
	});

	function toggleGroup(fingerprint: string) {
		expandedGroups = {
			...expandedGroups,
			[fingerprint]: !expandedGroups[fingerprint]
		};
	}
</script>

<div class="grid grid-cols-1 gap-4 lg:grid-cols-[260px_1fr]">
	<div class="hidden lg:block">
		<IssueFilterSidebar
			{report}
			{activeScanners}
			{activeSeverities}
			{activePage}
			{activeCategory}
			{searchTerm}
			{onScannersChange}
			{onSeveritiesChange}
			{onPageChange}
			{onCategoryChange}
			{onSearchChange}
			onClearAll={onClearFilters}
		/>
	</div>

	<div class="space-y-3">
		<div class="flex flex-wrap items-center gap-2 lg:hidden">
			<button
				type="button"
				onclick={() => (mobileFiltersOpen = !mobileFiltersOpen)}
				class="border-line bg-surface text-ink inline-flex items-center gap-2 rounded-lg border px-3 py-2 text-xs font-semibold"
				data-testid="open-mobile-filters"
			>
				<Filter class="h-3.5 w-3.5" /> Filters{activeFilterCount > 0
					? ` (${activeFilterCount})`
					: ''}
			</button>
		</div>

		{#if mobileFiltersOpen}
			<div class="lg:hidden">
				<IssueFilterSidebar
					{report}
					{activeScanners}
					{activeSeverities}
					{activePage}
					{activeCategory}
					{searchTerm}
					{onScannersChange}
					{onSeveritiesChange}
					{onPageChange}
					{onCategoryChange}
					{onSearchChange}
					onClearAll={onClearFilters}
				/>
			</div>
		{/if}

		<Panel class="ring-line/70 shadow-sm ring-1" padding="none" rounded="2xl">
			<div class="border-line border-b p-4 sm:p-5">
				<div class="flex flex-col gap-3">
					<div class="flex items-center justify-between gap-3">
						<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
							<span class="font-mono">{groups.length.toLocaleString()}</span>
							pattern{groups.length === 1 ? '' : 's'}
							<span class="text-ink-faint">·</span>
							<span class="font-mono">{filteredIssues.length.toLocaleString()}</span
							>{filteredIssues.length !== report.issues.length
								? ` / ${report.issues.length.toLocaleString()}`
								: ''} occurrence{filteredIssues.length === 1 ? '' : 's'}
						</h3>
						{#if hasActiveFilters}
							<button
								onclick={onClearFilters}
								class="text-accent text-xs font-semibold tracking-wide uppercase hover:underline"
							>
								Clear filters ({activeFilterCount})
							</button>
						{/if}
					</div>

					{#if hasActiveFilters}
						<div class="flex flex-wrap items-center gap-1.5 text-xs">
							{#each activeScanners as scannerId (scannerId)}
								<button
									type="button"
									onclick={() => onScannersChange(activeScanners.filter((s) => s !== scannerId))}
									class="border-line bg-surface-muted text-ink-muted hover:text-ink inline-flex items-center gap-1 rounded-full border px-2 py-1"
								>
									scanner: {scannerId}
									<X class="h-3 w-3" aria-hidden="true" />
								</button>
							{/each}
							{#each activeSeverities as severity (severity)}
								<button
									type="button"
									onclick={() => onSeveritiesChange(activeSeverities.filter((s) => s !== severity))}
									class="border-line bg-surface-muted text-ink-muted hover:text-ink inline-flex items-center gap-1 rounded-full border px-2 py-1"
								>
									severity: {severity}
									<X class="h-3 w-3" aria-hidden="true" />
								</button>
							{/each}
							{#if activeCategory}
								<button
									type="button"
									onclick={() => onCategoryChange(null)}
									class="border-line bg-surface-muted text-ink-muted hover:text-ink inline-flex items-center gap-1 rounded-full border px-2 py-1"
								>
									category: {activeCategory}
									<X class="h-3 w-3" aria-hidden="true" />
								</button>
							{/if}
							{#if activePage}
								<button
									type="button"
									onclick={() => onPageChange(null)}
									class="border-line bg-surface-muted text-ink-muted hover:text-ink inline-flex items-center gap-1 rounded-full border px-2 py-1"
								>
									page: {activePageLabel}
									<X class="h-3 w-3" aria-hidden="true" />
								</button>
							{/if}
							{#if searchTerm.trim()}
								<button
									type="button"
									onclick={() => onSearchChange('')}
									class="border-line bg-surface-muted text-ink-muted hover:text-ink inline-flex items-center gap-1 rounded-full border px-2 py-1"
								>
									search: "{searchTerm}"
									<X class="h-3 w-3" aria-hidden="true" />
								</button>
							{/if}
						</div>
					{/if}

					<div class="flex flex-wrap items-center justify-between gap-3">
						<div
							class="border-line/70 bg-surface-muted inline-flex items-center gap-1 rounded-lg border p-1 text-xs"
							role="tablist"
							aria-label="Issue view mode"
						>
							<button
								type="button"
								role="tab"
								aria-selected={viewMode === 'grouped'}
								onclick={() => (viewMode = 'grouped')}
								class={cn(
									'rounded px-2 py-1 font-semibold transition',
									viewMode === 'grouped' ? 'bg-surface text-ink shadow-xs' : 'text-ink-muted'
								)}
							>
								Patterns
							</button>
							<button
								type="button"
								role="tab"
								aria-selected={viewMode === 'flat'}
								onclick={() => (viewMode = 'flat')}
								class={cn(
									'rounded px-2 py-1 font-medium transition',
									viewMode === 'flat' ? 'bg-surface text-ink shadow-xs' : 'text-ink-faint'
								)}
							>
								Flat list
							</button>
						</div>
						<div class="flex items-center gap-2">
							<label
								for="issue-sort"
								class="text-ink-faint text-xs font-semibold tracking-wide uppercase"
							>
								Sort
							</label>
							<Select
								id="issue-sort"
								uiSize="sm"
								class="min-w-40"
								value={issueSort}
								onchange={(event) => onSortChange((event.currentTarget as HTMLSelectElement).value)}
							>
								{#each ISSUE_SORTS as sort (sort)}
									<option value={sort}>{ISSUE_SORT_LABELS[sort]}</option>
								{/each}
							</Select>
						</div>
					</div>

					<div
						class="border-line/70 bg-surface flex flex-wrap items-center justify-between gap-2 rounded-xl border px-2 py-2"
					>
						<div class="text-ink-muted text-xs">
							{#if viewMode === 'grouped'}
								<span class="font-mono">{actionableGroups.length.toLocaleString()}</span>
								actionable pattern{actionableGroups.length === 1 ? '' : 's'}
								{#if manualGroups.length > 0}
									· <span class="font-mono">{manualGroups.length.toLocaleString()}</span> manual review
								{/if}
								· <span class="font-mono">{sortedIssues.length.toLocaleString()}</span> occurrences
							{:else}
								<span class="font-mono">{sortedIssues.length.toLocaleString()}</span> visible results
							{/if}
						</div>
						<div
							class="border-line/70 bg-surface-muted inline-flex items-center gap-1 rounded-lg border p-1 text-xs"
						>
							<button
								type="button"
								onclick={() => (previewMode = 'auto')}
								class={cn(
									'rounded px-2 py-1 font-semibold transition',
									previewMode === 'auto' ? 'bg-surface text-ink shadow-xs' : 'text-ink-muted'
								)}
							>
								Previews auto
							</button>
							<button
								type="button"
								onclick={() => (previewMode = 'on')}
								class={cn(
									'rounded px-2 py-1 font-semibold transition',
									previewMode === 'on' ? 'bg-surface text-ink shadow-xs' : 'text-ink-muted'
								)}
							>
								On
							</button>
							<button
								type="button"
								onclick={() => (previewMode = 'off')}
								class={cn(
									'rounded px-2 py-1 font-semibold transition',
									previewMode === 'off' ? 'bg-surface text-ink shadow-xs' : 'text-ink-muted'
								)}
							>
								Off
							</button>
						</div>
					</div>
				</div>
			</div>

			{#if filteredIssues.length === 0}
				<div
					class="bg-surface-muted/30 flex flex-col items-center justify-center py-14 text-center"
				>
					<p class="text-ink font-medium">No issues found</p>
					<p class="text-ink-muted mt-1 text-sm">
						{#if hasActiveFilters}
							Try adjusting your filters
						{:else}
							This scan completed with no issues
						{/if}
					</p>
				</div>
			{:else if viewMode === 'grouped'}
				<div class="border-line/70 border-t" data-testid="issue-group-list">
					{#each actionableGroups as group (group.fingerprint)}
						<IssueGroupRow
							{group}
							{pagesById}
							{screenshots}
							{showPreviews}
							expanded={!!expandedGroups[group.fingerprint]}
							{selectedIssueId}
							onToggle={() => toggleGroup(group.fingerprint)}
							{onIssueSelect}
						/>
					{/each}

					{#if manualGroups.length > 0}
						<div class="bg-surface-muted/30" data-testid="manual-review-bucket">
							<button
								type="button"
								onclick={() => (manualOpen = !manualOpen)}
								aria-expanded={manualExpanded}
								class="hover:bg-surface-muted/70 border-line/70 flex w-full items-center gap-3 border-t px-4 py-3 text-left transition sm:px-5"
							>
								<span class="text-ink-faint flex h-4 w-4 shrink-0 items-center justify-center">
									{#if manualExpanded}
										<ChevronDown class="h-4 w-4" />
									{:else}
										<ChevronRight class="h-4 w-4" />
									{/if}
								</span>
								<Eye class="text-ink-faint h-4 w-4 shrink-0" aria-hidden="true" />
								<span class="min-w-0 flex-1">
									<span class="text-ink text-sm font-semibold">
										Needs manual review ({manualGroups.length.toLocaleString()})
									</span>
									<span class="text-ink-muted ml-2 text-xs">
										{manualCheckCount.toLocaleString()} check{manualCheckCount === 1 ? '' : 's'} the scanners
										couldn't verify automatically
									</span>
								</span>
							</button>
							{#if manualExpanded}
								<div class="border-line/60 border-t">
									{#each manualGroups as group (group.fingerprint)}
										<IssueGroupRow
											{group}
											{pagesById}
											{screenshots}
											{showPreviews}
											expanded={!!expandedGroups[group.fingerprint]}
											{selectedIssueId}
											onToggle={() => toggleGroup(group.fingerprint)}
											{onIssueSelect}
										/>
									{/each}
								</div>
							{/if}
						</div>
					{/if}
				</div>
			{:else}
				<div
					bind:this={listContainer}
					class={cn(
						'divide-line border-line/70 divide-y border-t',
						shouldVirtualize && 'max-h-[70vh] overflow-y-auto overscroll-contain'
					)}
					data-testid="issue-list"
					onscroll={handleListScroll}
				>
					{#if shouldVirtualize}
						<div style={`height: ${totalHeight}px; position: relative;`}>
							<div style={`transform: translateY(${offsetY}px);`}>
								{#each visibleIssues as issue (issue.id)}
									<IssueRowCard
										{issue}
										page={pagesById[issue.pageId] ?? null}
										{screenshots}
										showScreenshot={showPreviews}
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
								showScreenshot={showPreviews}
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
</div>
