<script lang="ts">
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { Input, Select } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { Search, X } from 'lucide-svelte';

	interface Props {
		report: UnifiedReport;
		activeScanners: string[];
		activeSeverities: string[];
		activePage: string | null;
		activeCategory: string | null;
		searchTerm: string;
		onScannersChange: (next: string[]) => void;
		onSeveritiesChange: (next: string[]) => void;
		onPageChange: (page: string | null) => void;
		onCategoryChange: (category: string | null) => void;
		onSearchChange: (query: string) => void;
		onClearAll: () => void;
		class?: string;
	}

	let {
		report,
		activeScanners,
		activeSeverities,
		activePage,
		activeCategory,
		searchTerm,
		onScannersChange,
		onSeveritiesChange,
		onPageChange,
		onCategoryChange,
		onSearchChange,
		onClearAll,
		class: className
	}: Props = $props();

	const SEVERITIES = ['critical', 'serious', 'moderate', 'minor', 'info'] as const;

	const severityCounts = $derived(
		report.summary.bySeverity ?? {
			critical: 0,
			serious: 0,
			moderate: 0,
			minor: 0,
			info: 0
		}
	);

	const scannerCounts = $derived(report.summary.byScanner ?? {});

	const categories = $derived.by(() => {
		const seen = new Set<string>();
		for (const issue of report.issues) {
			if (issue.category) seen.add(issue.category);
		}
		return Array.from(seen).sort();
	});

	const SEARCH_DEBOUNCE_MS = 250;
	let localSearch = $state('');
	$effect(() => {
		localSearch = searchTerm;
	});

	$effect(() => {
		const value = localSearch;
		if (value === searchTerm) return;
		const timer = setTimeout(() => onSearchChange(value.trim()), SEARCH_DEBOUNCE_MS);
		return () => clearTimeout(timer);
	});

	function toggleScanner(id: string) {
		const next = activeScanners.includes(id)
			? activeScanners.filter((s) => s !== id)
			: [...activeScanners, id];
		onScannersChange(next);
	}

	function toggleSeverity(id: string) {
		const next = activeSeverities.includes(id)
			? activeSeverities.filter((s) => s !== id)
			: [...activeSeverities, id];
		onSeveritiesChange(next);
	}

	const hasActive = $derived(
		activeScanners.length > 0 ||
			activeSeverities.length > 0 ||
			activePage !== null ||
			activeCategory !== null ||
			searchTerm.trim().length > 0
	);
</script>

<aside
	class={cn(
		'border-line bg-surface flex flex-col gap-5 rounded-2xl border p-4 text-sm shadow-sm',
		className
	)}
	aria-label="Issue filters"
	data-testid="issue-filter-sidebar"
>
	<div class="flex items-center justify-between">
		<h3 class="text-ink text-sm font-semibold tracking-tight">Filters</h3>
		{#if hasActive}
			<button
				type="button"
				onclick={onClearAll}
				class="text-accent inline-flex items-center gap-1 text-xs font-semibold hover:underline"
			>
				<X class="h-3 w-3" /> Clear
			</button>
		{/if}
	</div>

	<div>
		<label for="issue-sidebar-search" class="sr-only">Search issues</label>
		<div class="relative">
			<Search class="text-ink-faint absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2" />
			<Input
				id="issue-sidebar-search"
				value={localSearch}
				placeholder="Search issues"
				class="h-9 pl-8 text-xs"
				oninput={(event) =>
					(localSearch = (event.currentTarget as HTMLInputElement).value)}
			/>
		</div>
	</div>

	<fieldset class="flex flex-col gap-1.5">
		<legend class="text-ink-faint mb-1 text-[11px] font-semibold uppercase tracking-wide">
			Scanner
		</legend>
		{#each report.scanners as scanner (scanner.id)}
			{@const count = scannerCounts[scanner.id] ?? 0}
			{@const checked = activeScanners.includes(scanner.id)}
			<label
				class="hover:bg-surface-muted/60 flex cursor-pointer items-center justify-between gap-2 rounded-md px-1.5 py-1"
			>
				<span class="flex items-center gap-2 truncate">
					<input
						type="checkbox"
						{checked}
						onchange={() => toggleScanner(scanner.id)}
						class="border-line text-accent focus:ring-accent h-3.5 w-3.5 rounded"
					/>
					<span class="text-ink truncate text-xs">{scanner.name ?? scanner.id}</span>
				</span>
				<span class="text-ink-faint font-mono text-[11px]">{count.toLocaleString()}</span>
			</label>
		{/each}
	</fieldset>

	<fieldset class="flex flex-col gap-1.5">
		<legend class="text-ink-faint mb-1 text-[11px] font-semibold uppercase tracking-wide">
			Severity
		</legend>
		{#each SEVERITIES as severity (severity)}
			{@const count = severityCounts[severity] ?? 0}
			{@const checked = activeSeverities.includes(severity)}
			<label
				class="hover:bg-surface-muted/60 flex cursor-pointer items-center justify-between gap-2 rounded-md px-1.5 py-1"
			>
				<span class="flex items-center gap-2">
					<input
						type="checkbox"
						{checked}
						disabled={count === 0}
						onchange={() => toggleSeverity(severity)}
						class="border-line text-accent focus:ring-accent h-3.5 w-3.5 rounded"
					/>
					<span class="text-ink text-xs capitalize">{severity}</span>
				</span>
				<span class="text-ink-faint font-mono text-[11px]">{count.toLocaleString()}</span>
			</label>
		{/each}
	</fieldset>

	<div class="flex flex-col gap-1.5">
		<label for="issue-sidebar-page" class="text-ink-faint text-[11px] font-semibold uppercase tracking-wide">
			Page
		</label>
		<Select
			id="issue-sidebar-page"
			uiSize="sm"
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
	</div>

	{#if categories.length > 0}
		<div class="flex flex-col gap-1.5">
			<label for="issue-sidebar-category" class="text-ink-faint text-[11px] font-semibold uppercase tracking-wide">
				Category
			</label>
			<Select
				id="issue-sidebar-category"
				uiSize="sm"
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
		</div>
	{/if}
</aside>
