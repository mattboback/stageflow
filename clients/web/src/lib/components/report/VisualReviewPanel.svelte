<script lang="ts">
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, UnifiedReport } from '$lib/types/unified-report';

	import { Panel, SeverityBar } from '$lib/components/ui';
	import {
		getPageOverviewUrl,
		getSeverityContainerClass,
		getSeverityDotClass,
		getSeverityFillColor,
		getSeverityStrokeColor
	} from '$lib/report';
	import { cn, formatTimestamp } from '$lib/utils';
	import {
		ExternalLink,
		Printer,
		RotateCcw,
		ZoomIn,
		ZoomOut,
		Target,
		Focus,
		Columns,
		Maximize2
	} from 'lucide-svelte';
	import { tick } from 'svelte';

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
		Object.fromEntries(pageIssues.map((issue) => [issue.id, issue])) as Record<string, IssueDetail>
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
	let cameraFocusLock = $state(true); // Auto-zoom target camera onto selected issue
	let isRightPanelCollapsed = $state(false);
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
		// Largest boxes first so the smallest, most specific marker paints on top and
		// stays clickable — a full-page document-level box must not swallow clicks meant
		// for the boxes beneath it.
		return elements
			.filter((el) => {
				const issue = issueMap[el.issueId];
				if (!issue) return false;
				const sev = issue.severity as keyof typeof severityFilters;
				return severityFilters[sev] ?? true;
			})
			.sort((a, b) => b.width * b.height - a.width * a.height);
	});

	// Issues that have at least one overlay element on the current page
	const issuesWithMarkers = $derived(new Set(overlayElements.map((el) => el.issueId)));

	// First overlay element matching the selected issue (used as camera focal point)
	const focusedElement = $derived.by(() => {
		if (!activeIssueId) return null;
		return overlayElements.find((el) => el.issueId === activeIssueId) ?? null;
	});

	// Dynamic viewBox that pans and zooms dynamically when an issue is selected
	const activeViewBox = $derived.by(() => {
		if (!canRenderOverlay || !pageWidth || !pageHeight) {
			return '0 0 0 0';
		}
		if (!focusedElement || !cameraFocusLock) {
			return `0 0 ${pageWidth} ${pageHeight}`;
		}

		// Add relative magnification padding
		const padX = Math.max(280, focusedElement.width * 2.2);
		const padY = Math.max(200, focusedElement.height * 2.2);

		// Center alignment target
		const targetX = focusedElement.x + focusedElement.width / 2 - padX / 2;
		const targetY = focusedElement.y + focusedElement.height / 2 - padY / 2;

		// Bounds clamp
		const x = Math.max(0, Math.min(pageWidth - padX, targetX));
		const y = Math.max(0, Math.min(pageHeight - padY, targetY));
		const w = Math.min(pageWidth, padX);
		const h = Math.min(pageHeight, padY);

		return `${x} ${y} ${w} ${h}`;
	});

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
			el?.scrollIntoView?.({ block: 'nearest', behavior: 'smooth' });
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

	// Selecting an issue locks the lens camera onto it (activeIssueId drives the zoom)
	// and opens the shared, URL-driven detail modal (owned by ReportShell via onIssueSelect),
	// so issues stay deep-linkable. The optional elementId targets a specific occurrence.
	function selectIssue(issue: IssueDetail, elementId?: string) {
		activeIssueId = issue.id;
		onIssueSelect(issue, elementId);
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
				{@const isActiveTab = selectedPage?.id === p.id}
				{@const tabOverview = p.pageOverview ?? null}
				{@const tabOverviewUrl =
					tabOverview && (tabOverview.pageWidth ?? 0) > 0
						? getPageOverviewUrl(
								screenshots,
								p.id,
								activeScanner ? [activeScanner, 'axe'] : ['axe']
							)
						: null}
				<button
					type="button"
					onclick={() => onSelectPage(p.id)}
					class={cn(
						'flex items-center gap-2.5 rounded-xl px-3 py-2 text-left text-xs font-semibold whitespace-nowrap transition-all duration-200',
						isActiveTab
							? 'bg-ink text-surface ring-ink/20 scale-102 shadow-xs ring-2'
							: 'text-ink-muted hover:bg-surface-muted hover:text-ink'
					)}
					title={p.url}
				>
					{#if tabOverviewUrl && tabOverview}
						<span
							class={cn(
								'border-line/60 bg-surface h-8 w-11 shrink-0 overflow-hidden rounded-md border',
								isActiveTab && 'border-surface/30'
							)}
						>
							<svg
								class="h-full w-full"
								viewBox={`0 0 ${tabOverview.pageWidth} ${Math.min(tabOverview.pageHeight, Math.round(tabOverview.pageWidth * 0.72))}`}
								preserveAspectRatio="xMidYMin slice"
								aria-hidden="true"
							>
								<image
									href={tabOverviewUrl}
									x="0"
									y="0"
									width={tabOverview.pageWidth}
									height={tabOverview.pageHeight}
								/>
							</svg>
						</span>
					{/if}
					<span class="flex min-w-0 flex-col items-start gap-0.5">
						<span class="block max-w-[140px] truncate">{p.path ?? p.url}</span>
						{#if count > 0}
							<span class={cn('text-[10px]', isActiveTab ? 'opacity-65' : 'text-ink-faint')}
								>{count} issues</span
							>
						{/if}
						{#if p.bySeverity && count > 0}
							<SeverityBar counts={p.bySeverity} height="sm" class="w-16" />
						{/if}
					</span>
				</button>
			{/each}
		</div>
	</div>

	<button
		type="button"
		onclick={() => window.print()}
		class="border-line text-ink hover:border-accent hover:text-accent bg-surface inline-flex shrink-0 items-center gap-2 rounded-xl border px-4 py-2 text-sm font-semibold shadow-xs transition-all hover:scale-102"
	>
		<Printer class="h-4 w-4" />
		Print / Save PDF
	</button>
</div>

<!-- ── Two-column layout ──────────────────────────────────────────────────── -->
<div
	class={cn(
		'items-start gap-4 lg:grid print:hidden',
		isRightPanelCollapsed ? 'lg:grid-cols-1' : 'lg:grid-cols-[1fr_380px]'
	)}
>
	<!-- LEFT: Screenshot + overlay -->
	<Panel class="border-line overflow-hidden border shadow-sm" padding="none" rounded="2xl">
		<!-- Controls bar -->
		<div
			class="border-line bg-surface-muted/50 flex flex-wrap items-center gap-2 border-b px-4 py-2.5"
		>
			<!-- Severity filter toggles -->
			<div class="flex flex-wrap items-center gap-1.5">
				{#each Object.entries(severityFilters) as [key, enabled] (key)}
					<button
						type="button"
						onclick={() => toggleSeverity(key as keyof typeof severityFilters)}
						class={cn(
							'rounded-lg border px-2.5 py-1 text-[9px] font-bold tracking-wider uppercase transition-all duration-250',
							enabled
								? getSeverityContainerClass(key)
								: 'border-line text-ink-muted hover:bg-surface-muted'
						)}
					>
						{key}
					</button>
				{/each}
			</div>

			<!-- Zoom & Camera controls -->
			<div class="ml-auto flex items-center gap-2">
				<!-- Expand Screenshot Toggle -->
				<button
					type="button"
					onclick={() => (isRightPanelCollapsed = !isRightPanelCollapsed)}
					class={cn(
						'flex items-center gap-1 rounded-lg border px-2.5 py-1 text-xs font-semibold transition-all duration-200',
						isRightPanelCollapsed
							? 'bg-accent/15 border-accent text-accent shadow-xs'
							: 'border-line text-ink-muted hover:bg-surface-muted'
					)}
					title={isRightPanelCollapsed ? 'Show Issues List' : 'Expand Screenshot (Hide Issues)'}
				>
					{#if isRightPanelCollapsed}
						<Columns class="h-3.5 w-3.5" />
						<span class="text-[10px] font-bold tracking-wide uppercase">Split View</span>
					{:else}
						<Maximize2 class="h-3.5 w-3.5" />
						<span class="text-[10px] font-bold tracking-wide uppercase">Expand</span>
					{/if}
				</button>

				<div class="bg-line/80 mx-0.5 h-4 w-[1px]"></div>

				<!-- Auto Lens Focus Target Toggle -->
				<button
					type="button"
					onclick={() => (cameraFocusLock = !cameraFocusLock)}
					class={cn(
						'flex items-center gap-1 rounded-lg border px-2.5 py-1 text-xs font-semibold transition-all duration-200',
						cameraFocusLock
							? 'bg-accent/15 border-accent text-accent shadow-xs'
							: 'border-line text-ink-muted hover:bg-surface-muted'
					)}
					title="Auto Lens Target Zoom"
				>
					<Target class="h-3.5 w-3.5" />
					<span class="text-[10px] font-bold tracking-wide uppercase">Lens Lock</span>
				</button>

				<div class="bg-line/80 mx-0.5 h-4 w-[1px]"></div>

				<button
					type="button"
					onclick={() => adjustZoom(-0.25)}
					class="border-line text-ink-muted hover:bg-surface-muted rounded-lg border p-1 transition"
					title="Zoom out"
				>
					<ZoomOut class="h-3.5 w-3.5" />
				</button>
				<span
					class="text-ink-muted min-w-[2.5rem] text-center font-mono text-xs font-bold tabular-nums"
				>
					{Math.round(zoom * 100)}%
				</span>
				<button
					type="button"
					onclick={() => adjustZoom(0.25)}
					class="border-line text-ink-muted hover:bg-surface-muted rounded-lg border p-1 transition"
					title="Zoom in"
				>
					<ZoomIn class="h-3.5 w-3.5" />
				</button>
				<button
					type="button"
					onclick={resetZoom}
					class="border-line text-ink-muted hover:bg-surface-muted rounded-lg border p-1 transition"
					title="Reset zoom"
				>
					<RotateCcw class="h-3.5 w-3.5" />
				</button>
			</div>
		</div>

		<!-- Screenshot with SVG overlay -->
		{#if !overviewUrl}
			<div class="text-ink-muted p-12 text-center text-sm italic">
				No page overview screenshot is available for this page.
			</div>
		{:else if canRenderOverlay}
			<div class="overflow-auto bg-neutral-900 shadow-inner" style="max-height: 80vh">
				<div
					style="width: {zoom *
						100}%; will-change: width; transition: width 0.35s cubic-bezier(0.16, 1, 0.3, 1);"
				>
					<button
						type="button"
						class="block w-full bg-transparent p-0 text-left focus:outline-none"
						aria-label="Page screenshot with clickable issue markers"
					>
						<svg
							class="screenshot-lens-svg block w-full shadow-2xl"
							viewBox={activeViewBox}
							preserveAspectRatio="xMidYMid meet"
							role="img"
							aria-label="Page overview with markers"
						>
							<image href={overviewUrl} x="0" y="0" width={pageWidth} height={pageHeight} />
							{#each overlayElements as element (element.issueId + '|' + element.selector)}
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
										class="overlay-rect cursor-pointer"
										style="--hover-fill: {fillColor}; transition: all 0.25s;"
										role="button"
										tabindex="0"
										aria-label="{issue.title} ({issue.severity})"
										aria-pressed={isActive}
										onclick={() => selectIssue(issue, `${issue.id}-el-${element.nodeIndex}`)}
										onkeydown={(e) => {
											if (e.key === 'Enter' || e.key === ' ') {
												e.preventDefault();
												selectIssue(issue, `${issue.id}-el-${element.nodeIndex}`);
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
			<div class="overflow-auto bg-neutral-900 shadow-inner" style="max-height: 80vh">
				<div
					style="width: {zoom *
						100}%; will-change: width; transition: width 0.35s cubic-bezier(0.16, 1, 0.3, 1);"
				>
					<img
						src={overviewUrl}
						alt="Page screenshot"
						class="block w-full shadow-2xl"
						loading="lazy"
					/>
				</div>
			</div>
		{/if}

		<!-- Footer hint -->
		{#if overlayElements.length > 0}
			<div class="border-line bg-surface-muted/30 flex items-center gap-1.5 border-t px-4 py-2">
				<Focus class="text-accent h-3.5 w-3.5 animate-pulse" />
				<p class="text-ink-faint text-xs font-medium">
					{overlayElements.length} element{overlayElements.length !== 1 ? 's' : ''} targetable · select
					a marker box to target camera lens
				</p>
			</div>
		{/if}
	</Panel>

	<!-- RIGHT: Issues list -->
	{#if !isRightPanelCollapsed}
		<div>
			<Panel class="border-line overflow-hidden border shadow-sm" padding="none" rounded="2xl">
				<!-- Header -->
				<div class="border-line bg-surface-muted/20 border-b px-4 py-3">
					<h3 class="text-ink truncate text-sm font-semibold" title={selectedPage?.url}>
						{selectedPage?.path ?? selectedPage?.url ?? 'Page issues'}
					</h3>
					<p class="text-ink-muted mt-0.5 text-xs font-medium">
						{pageIssues.length} issue{pageIssues.length !== 1 ? 's' : ''} detected
						{#if activeIssueId}
							·
							<button
								type="button"
								class="text-accent font-bold hover:underline"
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
					class="divide-line divide-y overflow-y-auto"
					style="max-height: calc(80vh - 3.5rem)"
				>
					{#if pageIssues.length === 0}
						<div class="text-ink-muted p-10 text-center text-sm italic">
							No issues detected on this page.
						</div>
					{:else}
						{#each pageIssues as issue (issue.id)}
							{@const isActive = activeIssueId === issue.id}
							{@const hasMarker = issuesWithMarkers.has(issue.id)}
							<div class="group relative">
								<!-- Row: click to focus the lens on this issue and open its detail -->
								<div
									data-issue-id={issue.id}
									class={cn(
										'flex cursor-pointer items-start gap-2.5 border-l-4 px-4 py-3.5 pr-10 transition-all duration-200',
										isActive
											? 'bg-accent/5 border-accent scale-[1.002] shadow-xs'
											: 'hover:bg-surface-muted/60 border-transparent'
									)}
									role="button"
									tabindex="0"
									aria-pressed={isActive}
									onclick={() => selectIssue(issue)}
									onkeydown={(e) => {
										if (e.key === 'Enter' || e.key === ' ') {
											e.preventDefault();
											selectIssue(issue);
										}
									}}
								>
									<!-- Severity badge -->
									<span
										class={cn(
											'mt-0.5 inline-flex shrink-0 items-center gap-1 rounded border px-1.5 py-0.5 text-[9px] font-bold tracking-wider uppercase',
											getSeverityContainerClass(issue.severity)
										)}
									>
										<span
											class={cn(
												'h-2 w-2 rounded-full',
												getSeverityDotClass(issue.severity),
												issue.severity === 'critical' && 'animate-hotspot-pulse text-red-500'
											)}
										></span>
										{issue.severity}
									</span>

									<!-- Title + meta -->
									<div class="min-w-0 flex-1">
										<p class="text-ink line-clamp-2 text-sm leading-snug font-semibold">
											{issue.title}
										</p>
										<p class="text-ink-faint mt-1 text-[11px] font-medium">
											{issue.elementCount} element{issue.elementCount !== 1 ? 's' : ''}
											{#if hasMarker}
												·
												<span class="text-accent font-semibold">interactive marker</span>
											{/if}
										</p>
										{#if isActive}
											<p class="text-accent mt-1.5 text-[10px] font-bold tracking-wide uppercase">
												Locked · lens focused on this occurrence
											</p>
										{/if}
									</div>
								</div>

								<!-- Details icon: always visible on hover, stops propagation -->
								<button
									type="button"
									onclick={(e) => {
										e.stopPropagation();
										selectIssue(issue);
									}}
									class="text-ink-faint hover:text-accent focus-visible:text-accent absolute top-4 right-3.5 rounded p-0.5 opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
									title="View full details"
									aria-label="View full details for {issue.title}"
								>
									<ExternalLink class="h-3.5 w-3.5" />
								</button>
							</div>
						{/each}
					{/if}
				</div>
			</Panel>

			{#if activeIssueId && issuesWithMarkers.has(activeIssueId)}
				<p
					class="text-ink-muted text-accent/80 mt-2 text-center text-xs font-semibold tracking-wide uppercase"
				>
					← Focal coordinates locked on active occurrence
				</p>
			{/if}
		</div>
	{/if}
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
							<th class="border border-gray-200 px-2 py-1.5 text-left font-semibold">Severity</th>
							<th class="border border-gray-200 px-2 py-1.5 text-left font-semibold">Issue</th>
							<th class="border border-gray-200 px-2 py-1.5 text-left font-semibold">Scanner</th>
							<th class="border border-gray-200 px-2 py-1.5 text-center font-semibold">Elements</th>
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
								<td class="border border-gray-200 px-2 py-1.5 text-center">{issue.elementCount}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	{/each}
</div>

<style>
	/* SVG hover fill for overlay boxes — CSS custom property since Tailwind doesn't work in SVG */
	.overlay-rect:hover {
		fill: var(--hover-fill);
		opacity: 0.15;
	}

	.screenshot-lens-svg {
		transition: viewBox 0.4s cubic-bezier(0.16, 1, 0.3, 1) !important;
		will-change: viewBox;
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
