<script lang="ts">
import type { ScreenshotArtifact } from "$lib/types/scan";
import type { IssueDetail, UnifiedReport } from "$lib/types/unified-report";

import { Input, Panel, Select, Tabs } from "$lib/components/ui";
import {
	ISSUE_SORTS,
	ISSUE_SORT_LABELS,
	filterIssues,
	getSeverityChipClass,
	getVirtualWindow,
	isIssueSortKey,
	sortIssues,
} from "$lib/report";
import { cn } from "$lib/utils";
import { Search } from "lucide-svelte";

import IssueRowCard from "./IssueRowCard.svelte";

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

let {
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
	onClearFilters,
}: Props = $props();

const severityOptions = [
	"all",
	"critical",
	"serious",
	"moderate",
	"minor",
	"info",
] as const;
const severityCounts = $derived(
	report.summary.bySeverity ?? {
		critical: 0,
		serious: 0,
		moderate: 0,
		minor: 0,
		info: 0,
	},
);

const scannerTabs = $derived.by(() => {
	const byScanner = report.summary.byScanner ?? {};
	return [
		{ id: "all", label: `All (${report.issues.length.toLocaleString()})` },
		...report.scanners.map((scanner) => ({
			id: scanner.id,
			label: `${scanner.name ?? scanner.id}${byScanner[scanner.id] ? ` (${byScanner[scanner.id].toLocaleString()})` : ""}`,
		})),
	];
});

const categories = $derived.by(() => {
	const seen: Record<string, true> = {};
	for (const issue of report.issues) {
		if (issue.category) seen[issue.category] = true;
	}
	return Object.keys(seen).sort();
});

const pagesById = $derived(
	Object.fromEntries(report.pages.map((page) => [page.id, page])),
);
const activePageLabel = $derived(
	activePage
		? (pagesById[activePage]?.path ?? pagesById[activePage]?.url ?? activePage)
		: null,
);
const filteredIssues = $derived(
	filterIssues(report.issues, {
		scannerId: activeScanner,
		pageId: activePage,
		severity: activeSeverity,
		category: activeCategory,
		query: searchTerm,
	}),
);

const sortedIssues = $derived(
	sortIssues(
		filteredIssues,
		isIssueSortKey(issueSort) ? issueSort : "severity",
	),
);

const hasActiveFilters = $derived(
	Boolean(
		activeScanner ||
			activePage ||
			activeSeverity ||
			activeCategory ||
			searchTerm.trim(),
	),
);

const hasSecondaryFilters = $derived(Boolean(activeCategory || activePage));
const activeFilterCount = $derived(
	(activeScanner ? 1 : 0) +
		(activePage ? 1 : 0) +
		(activeSeverity ? 1 : 0) +
		(activeCategory ? 1 : 0) +
		(searchTerm.trim() ? 1 : 0),
);

// Debounced search: local state syncs immediately for responsive UI,
// but URL update is debounced to avoid expensive recalculations on each keystroke.
// We use $state here instead of writable $derived because:
// 1. We need to sync from prop (searchTerm) when it changes externally
// 2. We need to allow local mutations from user input
// 3. We need to debounce before calling the callback
const SEARCH_DEBOUNCE_MS = 250;
// eslint-disable-next-line svelte/prefer-writable-derived -- complex bidirectional sync with debounce
let localSearchValue = $state("");
let showMoreFilters = $state(false);
let previewMode = $state<"auto" | "on" | "off">("auto");
let isCompactViewport = $state(false);

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
let severityChipScroller = $state<HTMLDivElement | null>(null);
let severitySwipeHintVisible = $state(false);
let scrollTop = $state(0);
let viewportHeight = $state(600);
let scrollRaf = $state<number | null>(null);

function updateSeveritySwipeHint() {
	if (!severityChipScroller || !isCompactViewport) {
		severitySwipeHintVisible = false;
		return;
	}
	const canScroll =
		severityChipScroller.scrollWidth - severityChipScroller.clientWidth > 8;
	const atStart = severityChipScroller.scrollLeft < 8;
	severitySwipeHintVisible = canScroll && atStart;
}

function handleSeverityChipScroll() {
	updateSeveritySwipeHint();
}

const showPreviews = $derived.by(() => {
	if (previewMode === "on") return true;
	if (previewMode === "off") return false;
	return !isCompactViewport;
});

const rowHeight = $derived(showPreviews ? 120 : 88);
const overscan = $derived(isCompactViewport ? 4 : 6);

const shouldVirtualize = $derived(sortedIssues.length > 200);
const totalHeight = $derived(sortedIssues.length * rowHeight);
const virtualWindow = $derived.by(() =>
	getVirtualWindow({
		scrollTop,
		viewportHeight,
		rowHeight,
		totalItems: sortedIssues.length,
		overScan: overscan,
	}),
);
const visibleIssues = $derived(
	shouldVirtualize
		? sortedIssues.slice(virtualWindow.startIndex, virtualWindow.endIndex)
		: sortedIssues,
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
	if (
		typeof window === "undefined" ||
		typeof window.matchMedia !== "function"
	) {
		isCompactViewport = false;
		return;
	}
	const media = window.matchMedia("(max-width: 640px)");
	const update = (event?: MediaQueryListEvent) => {
		isCompactViewport = event ? event.matches : media.matches;
	};
	update();
	media.addEventListener("change", update);
	return () => media.removeEventListener("change", update);
});

$effect(() => {
	if (!severityChipScroller) return;
	const frame = requestAnimationFrame(() => updateSeveritySwipeHint());
	if (typeof ResizeObserver === "undefined") {
		return () => cancelAnimationFrame(frame);
	}
	const observer = new ResizeObserver(() => updateSeveritySwipeHint());
	observer.observe(severityChipScroller);
	return () => {
		cancelAnimationFrame(frame);
		observer.disconnect();
	};
});

$effect(() => {
	if (!listContainer || typeof ResizeObserver === "undefined") return;
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
		if (scrollRaf !== null) {
			cancelAnimationFrame(scrollRaf);
		}
	};
});
</script>

<div class="space-y-3">
	<Tabs
		tabs={scannerTabs}
		value={activeScanner ?? 'all'}
		onValueChange={(id) => onScannerChange(id === 'all' ? null : id)}
	/>

	<Panel class="shadow-sm ring-1 ring-line/70" padding="none" rounded="2xl">
		<div class="border-line border-b p-4 sm:p-5">
			<div class="flex flex-col gap-4">
				<div class="flex items-center justify-between gap-3">
					<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
						Issues ({filteredIssues.length}{filteredIssues.length !== report.issues.length
							? ` / ${report.issues.length}`
							: ''})
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
						{#if activeScanner}
							<button
								type="button"
								onclick={() => onScannerChange(null)}
								class="border-line bg-surface-muted text-ink-muted hover:text-ink inline-flex items-center gap-1 rounded-full border px-2 py-1"
							>
								scanner: {activeScanner} <span aria-hidden="true">x</span>
							</button>
						{/if}
						{#if activeSeverity}
							<button
								type="button"
								onclick={() => onSeverityChange(null)}
								class="border-line bg-surface-muted text-ink-muted hover:text-ink inline-flex items-center gap-1 rounded-full border px-2 py-1"
							>
								severity: {activeSeverity} <span aria-hidden="true">x</span>
							</button>
						{/if}
						{#if activeCategory}
							<button
								type="button"
								onclick={() => onCategoryChange(null)}
								class="border-line bg-surface-muted text-ink-muted hover:text-ink inline-flex items-center gap-1 rounded-full border px-2 py-1"
							>
								category: {activeCategory} <span aria-hidden="true">x</span>
							</button>
						{/if}
						{#if activePage}
							<button
								type="button"
								onclick={() => onPageChange(null)}
								class="border-line bg-surface-muted text-ink-muted hover:text-ink inline-flex items-center gap-1 rounded-full border px-2 py-1"
							>
								page: {activePageLabel} <span aria-hidden="true">x</span>
							</button>
						{/if}
						{#if searchTerm.trim()}
							<button
								type="button"
								onclick={() => onSearchChange('')}
								class="border-line bg-surface-muted text-ink-muted hover:text-ink inline-flex items-center gap-1 rounded-full border px-2 py-1"
							>
								search: "{searchTerm}" <span aria-hidden="true">x</span>
							</button>
						{/if}
					</div>
				{/if}
				<div class="grid grid-cols-1 gap-3 md:grid-cols-[1fr,auto]">
					<div class="relative">
						<Search class="text-ink-faint absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2" />
						<Input
							value={localSearchValue}
							placeholder="Search issues"
							class="h-10 pl-9"
							oninput={(event) =>
								handleSearchInput((event.currentTarget as HTMLInputElement).value)}
						/>
					</div>
					<div class="flex items-center justify-start gap-2 md:justify-end">
						<label for="issue-sort" class="text-ink-faint text-xs font-semibold tracking-wide uppercase">
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
				<div class="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-line/70 bg-surface px-2 py-2">
					<div class="text-ink-muted text-xs">
						{sortedIssues.length.toLocaleString()} visible results
					</div>
					<div class="inline-flex items-center gap-1 rounded-lg border border-line/70 bg-surface-muted p-1 text-xs">
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
				<div class="bg-surface-muted/60 border-line rounded-xl border px-2 py-2.5">
					<div class="relative">
						<div
							bind:this={severityChipScroller}
							data-testid="severity-chip-scroller"
							onscroll={handleSeverityChipScroll}
							class="overflow-x-auto pb-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
						>
							<div class="flex min-w-max items-center gap-2 pr-5 sm:pr-0">
							{#each severityOptions as severity (severity)}
								{@const count = severity === 'all'
									? report.issues.length
									: severityCounts[severity] ?? 0}
								<button
									onclick={() => onSeverityChange(severity === 'all' ? null : severity)}
									disabled={severity !== 'all' && count === 0}
									class={getSeverityChipClass(
										severity,
										activeSeverity === null ? severity === 'all' : activeSeverity === severity
									)}
								>
									{severity} ({count.toLocaleString()})
								</button>
							{/each}
							</div>
						</div>
						{#if severitySwipeHintVisible}
							<div class="pointer-events-none absolute inset-y-0 right-0 w-8 bg-gradient-to-l from-surface-muted to-transparent sm:hidden"></div>
						{/if}
					</div>
					<div class="mt-1 flex items-center justify-between gap-2">
						{#if severitySwipeHintVisible}
							<span class="text-ink-faint text-[11px] font-medium sm:hidden">Swipe for more</span>
						{/if}
						<button
							type="button"
							onclick={() => {
								showMoreFilters = !showMoreFilters;
							}}
							class="text-ink-muted hover:text-ink ml-auto text-xs font-semibold transition"
						>
							{showMoreFilters || hasSecondaryFilters ? 'Less filters' : 'More filters'}
						</button>
					</div>
				</div>
				{#if showMoreFilters || hasSecondaryFilters}
					<div class="grid grid-cols-1 gap-3 border-t border-dashed border-line pt-3 sm:grid-cols-2">
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
				{/if}
			</div>
		</div>

		{#if filteredIssues.length === 0}
			<div class="bg-surface-muted/30 flex flex-col items-center justify-center py-14 text-center">
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
				class={cn(
					'divide-line divide-y border-t border-line/70',
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
