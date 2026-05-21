<script lang="ts">
	import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

	import { Chip, Panel } from '$lib/components/ui';
	import {
		getIssueKind,
		getSeverityFillColor,
		getSeverityStrokeColor,
		rewriteIssueTitle
	} from '$lib/report';
	import { cn } from '$lib/utils';

	interface Props {
		page: PageSummary;
		issues: IssueDetail[];
		screenshotUrl: string | null;
		onSelectIssue: (issue: IssueDetail, highlightedElementId?: string) => void;
	}

	let { page, issues, screenshotUrl, onSelectIssue }: Props = $props();

	const issueMap = $derived.by(
		() =>
			Object.fromEntries(issues.map((issue) => [issue.id, issue])) as Record<string, IssueDetail>
	);

	const elements = $derived(page.pageOverview?.elements ?? []);
	const pageWidth = $derived(page.pageOverview?.pageWidth ?? 0);
	const pageHeight = $derived(page.pageOverview?.pageHeight ?? 0);
	const canRenderOverview = $derived(Boolean(screenshotUrl) && pageWidth > 0 && pageHeight > 0);
	const hasOverlayElements = $derived(elements.length > 0);

	let zoom = $state(1);

	function clampZoom(value: number) {
		return Math.max(0.5, Math.min(3, value));
	}

	function adjustZoom(delta: number) {
		zoom = clampZoom(Math.round((zoom + delta) * 10) / 10);
	}

	function resetZoom() {
		zoom = 1;
	}

	let severityFilters = $state({
		critical: true,
		serious: true,
		moderate: true,
		minor: true,
		info: true
	});

	// Focus mode: only the hovered/focused element is rendered prominently;
	// non-focused boxes are dimmed. "Show all" turns this off.
	let focusMode = $state(true);

	function toggleSeverity(key: keyof typeof severityFilters) {
		severityFilters = { ...severityFilters, [key]: !severityFilters[key] };
	}

	const filteredElements = $derived(
		elements.filter((el) => {
			const issue = issueMap[el.issueId];
			if (!issue) return false;
			const severity = issue.severity as keyof typeof severityFilters;
			return severityFilters[severity] ?? true;
		})
	);

	let focusedIndex = $state(-1);
	let hoveredIndex = $state<number | null>(null);

	const activeIndex = $derived(hoveredIndex ?? (focusedIndex >= 0 ? focusedIndex : null));

	function handleKeydown(e: KeyboardEvent) {
		if (filteredElements.length === 0) return;

		if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
			e.preventDefault();
			focusedIndex = (focusedIndex + 1) % filteredElements.length;
		} else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
			e.preventDefault();
			focusedIndex = focusedIndex <= 0 ? filteredElements.length - 1 : focusedIndex - 1;
		} else if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			const el = filteredElements[focusedIndex];
			if (el) {
				const issue = issueMap[el.issueId];
				if (issue) {
					onSelectIssue(issue, `${issue.id}-el-${el.nodeIndex}`);
				}
			}
		}
	}
</script>

<div class="space-y-3">
	{#if canRenderOverview}
		<div class="text-ink-muted flex flex-wrap items-center justify-between gap-3 text-xs">
			{#if hasOverlayElements}
				<div class="flex flex-wrap items-center gap-2">
					<span class="font-semibold tracking-wide uppercase">Overlay filters</span>
					{#each Object.entries(severityFilters) as [key, enabled] (key)}
						<Chip
							as="button"
							type="button"
							caps
							interactive
							tone={enabled ? 'active' : 'default'}
							onclick={() => toggleSeverity(key as keyof typeof severityFilters)}
						>
							{key}
						</Chip>
					{/each}
					<span class="border-line/70 mx-1 h-4 border-l" aria-hidden="true"></span>
					<Chip
						as="button"
						type="button"
						caps
						interactive
						tone={focusMode ? 'active' : 'default'}
						onclick={() => (focusMode = !focusMode)}
						title={focusMode
							? 'Currently dimming non-selected boxes. Click to show all overlays.'
							: 'Currently showing all overlays. Click to focus only the hovered/selected box.'}
					>
						{focusMode ? 'Focus mode' : 'Show all'}
					</Chip>
				</div>
			{:else}
				<p class="text-ink-muted text-xs">No overlay markers were produced for this page.</p>
			{/if}

			<div class="flex items-center gap-2">
				<Chip
					as="button"
					type="button"
					interactive
					onclick={() => adjustZoom(-0.2)}
					title="Zoom out"
				>
					-
				</Chip>
				<span class="text-ink-muted tabular-nums">{Math.round(zoom * 100)}%</span>
				<Chip as="button" type="button" interactive onclick={() => adjustZoom(0.2)} title="Zoom in">
					+
				</Chip>
				<Chip as="button" type="button" interactive onclick={resetZoom} title="Reset zoom">
					Reset
				</Chip>
			</div>
		</div>
	{/if}

	{#if !screenshotUrl}
		<Panel variant="muted" padding="lg" rounded="2xl" class="text-ink-muted text-center text-sm">
			No page overview screenshot available for this scan.
		</Panel>
	{:else if canRenderOverview}
		<div class="grid grid-cols-1 gap-3 lg:grid-cols-[1fr,260px]">
			<Panel
				padding="none"
				rounded="2xl"
				class="relative overflow-hidden"
				data-testid="page-overview"
			>
				<div class="overflow-auto">
					<div
						class="relative"
						style={`transform: scale(${zoom}); transform-origin: top left; width: ${100 / zoom}%;`}
					>
						<button
							type="button"
							data-testid="page-overview-keyboard"
							class="block w-full bg-transparent p-0 text-left focus:outline-none focus-visible:ring-2 focus-visible:ring-teal-600 focus-visible:ring-offset-2"
							onkeydown={handleKeydown}
						>
							<svg
								class="block w-full"
								viewBox={`0 0 ${pageWidth} ${pageHeight}`}
								preserveAspectRatio="xMinYMin meet"
								role="img"
								aria-label={`Page overview for ${page.path ?? page.url} with ${filteredElements.length} highlighted issues`}
							>
								<image href={screenshotUrl} x="0" y="0" width={pageWidth} height={pageHeight} />
								{#each filteredElements as element, idx (`${element.issueId}:${element.nodeIndex}:${idx}`)}
									{@const issue = issueMap[element.issueId]}
									{#if issue}
										{@const strokeColor = getSeverityStrokeColor(issue.severity)}
										{@const fillColor = getSeverityFillColor(issue.severity)}
										{@const isActive = activeIndex === idx}
										{@const someActive = activeIndex !== null}
										{@const dimmed = focusMode && someActive && !isActive}
										{@const baseStroke = isActive ? 8 : focusMode && !someActive ? 2 : 4}
										{@const opacity = dimmed ? 0.18 : 1}
										<rect
											x={element.x}
											y={element.y}
											width={element.width}
											height={element.height}
											fill={isActive ? fillColor : 'transparent'}
											stroke={strokeColor}
											stroke-width={baseStroke}
											{opacity}
											class="cursor-pointer motion-safe:transition-[fill,stroke-width,opacity]"
											style={`--hover-fill: ${fillColor}`}
											role="button"
											tabindex="0"
											aria-label={`${rewriteIssueTitle(issue)} (${issue.severity})`}
											onmouseenter={() => (hoveredIndex = idx)}
											onmouseleave={() => (hoveredIndex = null)}
											onclick={() => onSelectIssue(issue, `${issue.id}-el-${element.nodeIndex}`)}
											onkeydown={(e) => {
												if (e.key === 'Enter' || e.key === ' ') {
													e.preventDefault();
													onSelectIssue(issue, `${issue.id}-el-${element.nodeIndex}`);
												}
											}}
											data-testid="page-overview-marker"
										>
											<title
												>{rewriteIssueTitle(issue)} — {issue.scanner} · {issue.ruleId} · {issue.severity}</title
											>
										</rect>
									{/if}
								{/each}
							</svg>

							{#if activeIndex !== null && filteredElements[activeIndex]}
								{@const el = filteredElements[activeIndex]}
								{@const issue = issueMap[el.issueId]}
								{#if issue}
									{@const labelX = el.x + el.width / 2}
									{@const labelY = Math.max(el.y - 32, 0)}
									{@const kind = getIssueKind(issue)}
									{@const labelText = `${rewriteIssueTitle(issue)}`}
									<div
										class="pointer-events-none absolute"
										style={`left: ${(labelX / pageWidth) * 100}%; top: ${(labelY / pageHeight) * 100}%; transform: translate(-50%, -100%);`}
									>
										<div
											class="border-ink/10 max-w-xs rounded-lg border bg-neutral-900/95 px-2.5 py-1.5 text-[11px] leading-snug text-white shadow-lg backdrop-blur-md"
										>
											<div class="font-semibold">{labelText}</div>
											<div class="mt-0.5 font-mono text-[10px] text-neutral-300">
												{issue.scanner} · {issue.ruleId} · {issue.severity} · {kind}
											</div>
										</div>
									</div>
								{/if}
							{/if}
						</button>
					</div>
				</div>
			</Panel>

			<aside
				class="border-line bg-surface flex max-h-[72vh] flex-col overflow-hidden rounded-2xl border shadow-sm"
				aria-label="Highlighted elements"
				data-testid="page-overview-element-sidebar"
			>
				<div class="border-line border-b px-3 py-2">
					<h4 class="text-ink text-xs font-semibold tracking-wide uppercase">
						Elements ({filteredElements.length})
					</h4>
					<p class="text-ink-muted mt-0.5 text-[11px]">
						{focusMode
							? 'Hover or click to highlight; non-selected boxes are dimmed.'
							: 'All boxes shown; hover or click to inspect.'}
					</p>
				</div>
				<ul class="divide-line/70 divide-y overflow-y-auto text-sm">
					{#each filteredElements as element, idx (`${element.issueId}:${element.nodeIndex}:${idx}`)}
						{@const issue = issueMap[element.issueId]}
						{#if issue}
							{@const isActive = activeIndex === idx}
							<li>
								<button
									type="button"
									onmouseenter={() => (hoveredIndex = idx)}
									onmouseleave={() => (hoveredIndex = null)}
									onfocus={() => (focusedIndex = idx)}
									onclick={() => onSelectIssue(issue, `${issue.id}-el-${element.nodeIndex}`)}
									class={cn(
										'hover:bg-surface-muted/60 flex w-full items-start gap-2 px-3 py-2 text-left transition',
										isActive && 'bg-surface-muted'
									)}
									data-testid="page-overview-element-row"
								>
									<span class="text-ink-faint mt-0.5 shrink-0 font-mono text-[11px]">
										#{idx + 1}
									</span>
									<span class="min-w-0 flex-1">
										<span class="text-ink block truncate text-xs font-semibold">
											{rewriteIssueTitle(issue)}
										</span>
										<span class="text-ink-faint mt-0.5 block text-[11px] capitalize">
											{issue.severity} · {issue.scanner}
										</span>
									</span>
								</button>
							</li>
						{/if}
					{/each}
				</ul>
			</aside>
		</div>
	{:else}
		<Panel variant="muted" padding="lg" rounded="2xl" class="text-ink-muted text-center text-sm">
			Page overview dimensions not available.
		</Panel>
	{/if}
</div>

<style>
	/* SVG hover fill using CSS custom property - Tailwind classes don't work reliably in SVG */
	rect[data-testid='page-overview-marker']:hover {
		fill: var(--hover-fill);
	}
</style>
