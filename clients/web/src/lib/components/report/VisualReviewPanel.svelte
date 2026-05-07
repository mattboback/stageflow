<script lang="ts">
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, UnifiedReport } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';
	import {
		getPageOverviewUrl,
		getSeverityBadgeClass,
		getSeverityFillColor,
		getSeverityStrokeColor
	} from '$lib/report';
	import { cn, formatTimestamp } from '$lib/utils';
	import { ExternalLink, Printer, RotateCcw, ZoomIn, ZoomOut } from 'lucide-svelte';
	import { tick } from 'svelte';

	import IssueDetailModal from './IssueDetailModal.svelte';

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

	// ─── Page selection ───────────────────────────────────────────────────────

	const selectedPage = $derived(
		report.pages.find((p) => p.id === activePage) ?? report.pages[0] ?? null
	);

	// ─── Issues ──────────────────────────────────────────────────────────────

	const issuesByPage = $derived.by(() => {
		const map: Record<string, IssueDetail[]> = {};
		for (const issue of report.issues) {
			if (activeScanner && issue.scanner !== activeScanner) continue;
			const list = map[issue.pageId] ?? [];
			list.push(issue);
			map[issue.pageId] = list;
		}
		return map;
	});

	const pageIssues = $derived.by(() => {
		const issues = selectedPage ? (issuesByPage[selectedPage.id] ?? []) : [];
		const seen = new Set<string>();
		return issues.filter((issue, i) => {
			const key = issue.id || String(i);
			if (seen.has(key)) return false;
			seen.add(key);
			return true;
		});
	});

	const issueMap = $derived(
		Object.fromEntries(pageIssues.map((issue) => [issue.id, issue])) as Record<
			string,
			IssueDetail
		>
	);

	// ─── Screenshot / overlay ─────────────────────────────────────────────────

	const overviewUrl = $derived(
		selectedPage
			? getPageOverviewUrl(
					screenshots,
					selectedPage.id,
					activeScanner ? [activeScanner, 'axe'] : ['axe']
				)
			: null
	);

	const pageWidth = $derived(selectedPage?.pageOverview?.pageWidth ?? 0);
	const pageHeight = $derived(selectedPage?.pageOverview?.pageHeight ?? 0);
	const canRenderOverlay = $derived(!!overviewUrl && pageWidth > 0 && pageHeight > 0);

	// ─── Interaction state ────────────────────────────────────────────────────

	let activeIssueId = $state<string | null>(null);
	let zoom = $state(1);
	let severityFilters = $state({
		critical: true,
		serious: true,
		moderate: true,
		minor: true,
		info: true
	});

	// Reset selection + zoom when selected page changes
	$effect(() => {
		void selectedPage?.id;
		activeIssueId = null;
		zoom = 1;
	});

	// ─── Overlay elements (filtered by severity) ──────────────────────────────

	const overlayElements = $derived.by(() => {
		const elements = selectedPage?.pageOverview?.elements ?? [];
		return elements.filter((el) => {
			const issue = issueMap[el.issueId];
			if (!issue) return false;
			const sev = issue.severity as keyof typeof severityFilters;
			return severityFilters[sev] ?? true;
		});
	});

	// Issues that have at least one overlay element on the current page
	const issuesWithMarkers = $derived(
		new Set(overlayElements.map((el) => el.issueId))
	);

	// ─── Right-panel scroll sync ──────────────────────────────────────────────

	let rightPanelRef = $state<HTMLDivElement | null>(null);

	$effect(() => {
		if (!activeIssueId || !rightPanelRef) return;
		let cancelled = false;
		void (async () => {
			await tick();
			if (cancelled) return;
			const el = rightPanelRef?.querySelector<HTMLElement>(
				`[data-issue-id="${CSS.escape(activeIssueId ?? '')}"]`
			);
			el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
		})();
		return () => {
			cancelled = true;
		};
	});

	// ─── Zoom helpers ─────────────────────────────────────────────────────────

	function clampZoom(v: number) {
		return Math.max(0.25, Math.min(4, Math.round(v * 10) / 10));
	}
	function adjustZoom(delta: number) {
		zoom = clampZoom(zoom + delta);
	}
	function resetZoom() {
		zoom = 1;
	}

	// ─── Event handlers ───────────────────────────────────────────────────────

	function toggleSeverity(key: keyof typeof severityFilters) {
		severityFilters = { ...severityFilters, [key]: !severityFilters[key] };
	}

	function handleBoxClick(issueId: string) {
		activeIssueId = activeIssueId === issueId ? null : issueId;
	}

	function handleIssueCardClick(issue: IssueDetail) {
		if (activeIssueId === issue.id) {
			// Second click on an already-selected card → open modal
			openModal(issue);
		} else {
			activeIssueId = issue.id;
		}
	}

	// ─── Detail modal ─────────────────────────────────────────────────────────

	let modalIssue = $state<IssueDetail | null>(null);

	function openModal(issue: IssueDetail) {
		modalIssue = issue;
		onIssueSelect(issue);
	}

	function closeModal() {
		modalIssue = null;
	}

	// ─── Print summary ────────────────────────────────────────────────────────

	const scannedAt = $derived(formatTimestamp(report.meta.scannedAt));
</script>

<!-- ── Page tabs + Print button ──────────────────────────────────────────── -->
<div class="mb-4 flex items-center gap-3 print:hidden">
	<div class="relative min-w-0 flex-1 overflow-x-auto">
		<div class="flex min-w-max gap-1">
			{#each report.pages as p (p.id)}
				{@const count = (issuesByPage[p.id] ?? []).length}
				<button
					type="button"
					onclick={() => onSelectPage(p.id)}
					class={cn(
						'flex flex-col items-start rounded-lg px-3 py-1.5 text-left text-xs font-semibold whitespace-nowrap transition',
						selectedPage?.id === p.id
							? 'bg-ink text-surface'
							: 'text-ink-muted hover:bg-surface-muted hover:text-ink'
					)}
					title={p.url}
				>
					<span class="block max-w-[140px] truncate">{p.path ?? p.url}</span>
					{#if count > 0}
						<span
							class={cn(
								'text-[10px]',
								selectedPage?.id === p.id ? 'opacity-60' : 'text-ink-faint'
							)}>{count} issues</span
						>
					{/if}
				</button>
			{/each}
		</div>
	</div>

	<button
		type="button"
		onclick={() => window.print()}
		class="border-line text-ink hover:border-accent hover:text-accent inline-flex shrink-0 items-center gap-2 rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors"
	>
		<Printer class="h-4 w-4" />
		Print / Save PDF
	</button>
</div>

<!-- ── Two-column layout ──────────────────────────────────────────────────── -->
<div class="items-start gap-4 print:hidden lg:grid lg:grid-cols-[1fr,380px]">
	<!-- LEFT: Screenshot + overlay -->
	<Panel class="overflow-hidden shadow-sm" padding="none" rounded="2xl">
		<!-- Controls bar -->
		<div class="border-line flex flex-wrap items-center gap-2 border-b px-3 py-2">
			<!-- Severity filter toggles -->
			<div class="flex flex-wrap items-center gap-1.5">
				{#each Object.entries(severityFilters) as [key, enabled] (key)}
					<button
						type="button"
						onclick={() => toggleSeverity(key as keyof typeof severityFilters)}
						class={cn(
							'rounded-md border px-2 py-0.5 text-[10px] font-semibold tracking-wide uppercase transition',
							enabled
								? getSeverityBadgeClass(key)
								: 'border-line text-ink-muted hover:bg-surface-muted'
						)}
					>
						{key}
					</button>
				{/each}
			</div>

			<!-- Zoom controls -->
			<div class="ml-auto flex items-center gap-1.5">
				<button
					type="button"
					onclick={() => adjustZoom(-0.25)}
					class="border-line text-ink-muted hover:bg-surface-muted rounded border p-1 transition"
					title="Zoom out"
				>
					<ZoomOut class="h-3.5 w-3.5" />
				</button>
				<span class="text-ink-muted min-w-[3rem] text-center text-xs tabular-nums">
					{Math.round(zoom * 100)}%
				</span>
				<button
					type="button"
					onclick={() => adjustZoom(0.25)}
					class="border-line text-ink-muted hover:bg-surface-muted rounded border p-1 transition"
					title="Zoom in"
				>
					<ZoomIn class="h-3.5 w-3.5" />
				</button>
				<button
					type="button"
					onclick={resetZoom}
					class="border-line text-ink-muted hover:bg-surface-muted rounded border p-1 transition"
					title="Reset zoom"
				>
					<RotateCcw class="h-3.5 w-3.5" />
				</button>
			</div>
		</div>

		<!-- Screenshot with SVG overlay -->
		{#if !overviewUrl}
			<div class="text-ink-muted p-8 text-center text-sm">
				No page overview screenshot is available for this page.
			</div>
		{:else if canRenderOverlay}
			<div class="overflow-auto" style="max-height: 72vh">
				<div
					style="transform: scale({zoom}); transform-origin: top left; width: {100 /
						zoom}%; will-change: transform;"
				>
					<button
						type="button"
						class="block w-full bg-transparent p-0 text-left focus:outline-none"
						aria-label="Page screenshot with clickable issue markers"
					>
						<svg
							class="block w-full"
							viewBox="0 0 {pageWidth} {pageHeight}"
							preserveAspectRatio="xMinYMin meet"
							role="img"
							aria-label="Page overview for {selectedPage?.path ??
								selectedPage?.url} with {overlayElements.length} highlighted issue{overlayElements.length !==
							1
								? 's'
								: ''}"
						>
							<image href={overviewUrl} x="0" y="0" width={pageWidth} height={pageHeight} />
							{#each overlayElements as element, elIdx (elIdx)}
								{@const issue = issueMap[element.issueId]}
								{#if issue}
									{@const strokeColor = getSeverityStrokeColor(issue.severity)}
									{@const fillColor = getSeverityFillColor(issue.severity)}
									{@const isActive = activeIssueId === issue.id}
									<rect
										x={element.x}
										y={element.y}
										width={element.width}
										height={element.height}
										fill={isActive ? fillColor : 'transparent'}
										stroke={strokeColor}
										stroke-width={isActive ? 6 : 3}
										class="overlay-rect cursor-pointer motion-safe:transition-[fill,stroke-width]"
										style="--hover-fill: {fillColor}"
										role="button"
										tabindex="0"
										aria-label="{issue.title} ({issue.severity})"
										aria-pressed={isActive}
										onclick={() => handleBoxClick(issue.id)}
										onkeydown={(e) => {
											if (e.key === 'Enter' || e.key === ' ') {
												e.preventDefault();
												handleBoxClick(issue.id);
											}
										}}
									>
										<title>{issue.title} ({issue.severity})</title>
									</rect>
								{/if}
							{/each}
						</svg>
					</button>
				</div>
			</div>
		{:else}
			<!-- Screenshot available but no overlay dimensions — show image only -->
			<div class="overflow-auto" style="max-height: 72vh">
				<img src={overviewUrl} alt="Page screenshot" class="block w-full" loading="lazy" />
			</div>
		{/if}

		<!-- Footer hint -->
		{#if overlayElements.length > 0}
			<div class="border-line border-t px-3 py-2">
				<p class="text-ink-faint text-xs">
					{overlayElements.length} element{overlayElements.length !== 1 ? 's' : ''} marked · click
					a box to highlight the issue in the list →
				</p>
			</div>
		{/if}
	</Panel>

	<!-- RIGHT: Issues list -->
	<div>
		<Panel class="overflow-hidden shadow-sm" padding="none" rounded="2xl">
			<!-- Header -->
			<div class="border-line border-b px-4 py-3">
				<h3 class="text-ink text-sm font-semibold">
					{selectedPage?.path ?? selectedPage?.url ?? 'Page issues'}
				</h3>
				<p class="text-ink-muted mt-0.5 text-xs">
					{pageIssues.length} issue{pageIssues.length !== 1 ? 's' : ''}
					{#if activeIssueId}
						·
						<button
							type="button"
							class="text-accent hover:underline"
							onclick={() => (activeIssueId = null)}
						>
							Clear selection
						</button>
					{/if}
				</p>
			</div>

			<!-- Issue list -->
			<div
				bind:this={rightPanelRef}
				class="overflow-y-auto"
				style="max-height: calc(72vh - 3.5rem)"
			>
				{#if pageIssues.length === 0}
					<div class="text-ink-muted p-6 text-center text-sm">No issues on this page.</div>
				{:else}
					<div class="divide-line divide-y">
						{#each pageIssues as issue (issue.id)}
							{@const isActive = activeIssueId === issue.id}
							{@const hasMarker = issuesWithMarkers.has(issue.id)}
							<div class="group relative">
								<!-- Row: click to select (second click opens modal) -->
								<div
									data-issue-id={issue.id}
									class={cn(
										'flex cursor-pointer items-start gap-2.5 px-4 py-3 pr-10 transition-colors',
										isActive
											? 'bg-accent/5 ring-accent/20 ring-1 ring-inset'
											: 'hover:bg-surface-muted/60'
									)}
									role="button"
									tabindex="0"
									aria-pressed={isActive}
									onclick={() => handleIssueCardClick(issue)}
									onkeydown={(e) => {
										if (e.key === 'Enter' || e.key === ' ') {
											e.preventDefault();
											handleIssueCardClick(issue);
										}
									}}
								>
									<!-- Severity badge -->
									<span
										class={cn(
											'mt-0.5 shrink-0 rounded px-1.5 py-0.5 text-[10px] font-semibold tracking-wide uppercase',
											getSeverityBadgeClass(issue.severity)
										)}
									>
										{issue.severity}
									</span>

									<!-- Title + meta -->
									<div class="min-w-0 flex-1">
										<p class="text-ink line-clamp-2 text-sm font-medium leading-snug">
											{issue.title}
										</p>
										<p class="text-ink-faint mt-0.5 text-xs">
											{issue.elementCount} element{issue.elementCount !== 1 ? 's' : ''}
											{#if hasMarker}
												·
												<span class="text-accent font-medium">marked</span>
											{/if}
										</p>
										{#if isActive}
											<p class="text-ink-muted mt-1 text-[11px]">
												Click again to view full details
											</p>
										{/if}
									</div>
								</div>

								<!-- Details icon: always visible on hover, stops propagation -->
								<button
									type="button"
									onclick={(e) => {
										e.stopPropagation();
										openModal(issue);
									}}
									class="text-ink-faint hover:text-accent focus-visible:text-accent absolute top-3 right-3 rounded p-0.5 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
									title="View full details"
									aria-label="View full details for {issue.title}"
								>
									<ExternalLink class="h-3.5 w-3.5" />
								</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</Panel>

		{#if activeIssueId && issuesWithMarkers.has(activeIssueId)}
			<p class="text-ink-muted mt-2 text-center text-xs">
				← highlighted boxes shown on the screenshot
			</p>
		{/if}
	</div>
</div>

<!-- ── Print-only view (all pages, linearised) ─────────────────────────── -->
<div class="hidden print:block">
	<!-- Report header -->
	<div class="mb-6 border-b border-gray-200 pb-4">
		<h1 class="text-xl font-bold text-gray-900">{report.meta.baseUrl ?? 'Scan Report'}</h1>
		<p class="mt-1 text-sm text-gray-500">
			{report.summary.totalIssues} issues across {report.summary.pagesScanned ?? 0} pages
			{#if scannedAt}&nbsp;· Scanned {scannedAt}{/if}
		</p>
	</div>

	{#each report.pages as p (p.id)}
		{@const pIssues = issuesByPage[p.id] ?? []}
		{@const pUrl = getPageOverviewUrl(screenshots, p.id, ['axe'])}
		{#if pIssues.length > 0}
			<div class="print-page mb-10">
				<h2 class="mb-3 text-base font-semibold text-gray-800">
					{p.path ?? p.url}
					<span class="ml-2 text-sm font-normal text-gray-500">({pIssues.length} issues)</span>
				</h2>

				{#if pUrl}
					<img
						src={pUrl}
						alt="Screenshot of {p.path ?? p.url}"
						class="mb-4 block max-w-full rounded border border-gray-200"
						style="max-height: 280px; object-fit: contain; object-position: top left;"
					/>
				{/if}

				<table class="w-full border-collapse text-xs">
					<thead>
						<tr class="bg-gray-100">
							<th class="border border-gray-200 px-2 py-1.5 text-left font-semibold"
								>Severity</th
							>
							<th class="border border-gray-200 px-2 py-1.5 text-left font-semibold">Issue</th>
							<th class="border border-gray-200 px-2 py-1.5 text-left font-semibold">Scanner</th>
							<th class="border border-gray-200 px-2 py-1.5 text-center font-semibold"
								>Elements</th
							>
						</tr>
					</thead>
					<tbody>
						{#each pIssues as issue (issue.id)}
							<tr class="even:bg-gray-50">
								<td class="border border-gray-200 px-2 py-1.5 font-medium capitalize">
									{issue.severity}
								</td>
								<td class="border border-gray-200 px-2 py-1.5">{issue.title}</td>
								<td class="border border-gray-200 px-2 py-1.5">{issue.scanner}</td>
								<td class="border border-gray-200 px-2 py-1.5 text-center"
									>{issue.elementCount}</td
								>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/each}
</div>

<!-- Detail modal (shared with existing report sections) -->
{#if modalIssue}
	<IssueDetailModal
		issue={modalIssue}
		page={report.pages.find((p) => p.id === modalIssue?.pageId) ?? null}
		{screenshots}
		onClose={closeModal}
	/>
{/if}

<style>
	/* SVG hover fill for overlay boxes — CSS custom property since Tailwind doesn't work in SVG */
	.overlay-rect:hover {
		fill: var(--hover-fill);
	}

	@media print {
		/* Hide the interactive UI entirely when printing */
		:global(body > *:not(.print-root)) {
			display: none !important;
		}

		.print-page {
			page-break-inside: avoid;
		}

		.print-page + .print-page {
			page-break-before: always;
		}
	}
</style>
