<script lang="ts">
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

	import { Chip, Modal, Tabs, panelVariants } from '$lib/components/ui';
	import {
		generateContextualFix,
		getIssueKind,
		getIssueKindLabel,
		getIssueScreenshotUrl,
		getPageOverviewUrl,
		getSeverityContainerClass,
		getSeverityDotClass,
		isManualReviewIssue,
		rewriteIssueTitle
	} from '$lib/report';
	import { cn, getWcagUnderstandingUrl, normalizeWcagTag } from '$lib/utils';
	import { ChevronLeft, ChevronRight, ExternalLink, FileText, Globe, XCircle } from 'lucide-svelte';
	import { tick } from 'svelte';

	import IssueEvidenceSection from './IssueEvidenceSection.svelte';
	import IssueOccurrenceCard from './IssueOccurrenceCard.svelte';

	interface Props {
		issue: IssueDetail;
		page?: PageSummary | null;
		issues?: IssueDetail[];
		screenshots: ScreenshotArtifact[];
		highlightedElementId?: string;
		onClose: () => void;
		onNavigate?: (issueId: string) => void;
	}

	let {
		issue,
		page = null,
		issues = [],
		screenshots,
		highlightedElementId,
		onClose,
		onNavigate
	}: Props = $props();

	const currentIndex = $derived(issues.findIndex((i) => i.id === issue.id));
	const totalCount = $derived(issues.length);
	const hasPrev = $derived(currentIndex > 0);
	const hasNext = $derived(currentIndex >= 0 && currentIndex < totalCount - 1);

	function navigate(delta: number) {
		if (!onNavigate || currentIndex < 0) return;
		const next = currentIndex + delta;
		if (next < 0 || next >= totalCount) return;
		const target = issues[next];
		if (target) onNavigate(target.id);
	}

	const issueKind = $derived(getIssueKind(issue));
	const isManual = $derived(isManualReviewIssue(issue));
	const displayTitle = $derived(rewriteIssueTitle(issue));

	const primaryOccurrence = $derived(issue.occurrences?.[0] ?? null);
	const occurrenceCount = $derived(issue.occurrences?.length ?? 0);
	const contextualFix = $derived(generateContextualFix(issue, primaryOccurrence));

	const alsoDetectedBy = $derived.by(() => {
		const raw = issue.scannerData?.alsoDetectedBy;
		return Array.isArray(raw)
			? raw.filter((v): v is string => typeof v === 'string' && v !== '')
			: [];
	});

	const screenshotUrl = $derived(
		getIssueScreenshotUrl({
			screenshots,
			scannerId: issue.scanner,
			issueId: issue.id,
			pageId: issue.pageId
		})
	);

	const pageOverviewUrl = $derived(
		page
			? getPageOverviewUrl(screenshots, page.id, issue.scanner ? [issue.scanner, 'axe'] : ['axe'])
			: null
	);

	let modalRef = $state<HTMLDivElement | null>(null);
	let localHighlightedElementId = $state<string | null>(null);

	const activeHighlightId = $derived(localHighlightedElementId ?? highlightedElementId ?? null);
	const openedFromOverlay = $derived(Boolean(highlightedElementId));
	let fullPageEvidenceOverride = $state<boolean | null>(null);
	const showFullPageEvidence = $derived(fullPageEvidenceOverride ?? !highlightedElementId);
	const shouldShowPageOverview = $derived(showFullPageEvidence || !openedFromOverlay);
	const pageOverviewExists = $derived(
		!!page &&
			!!pageOverviewUrl &&
			(page.pageOverview?.pageWidth ?? 0) > 0 &&
			(page.pageOverview?.pageHeight ?? 0) > 0
	);
	const pageOverviewRenderable = $derived(
		pageOverviewExists && (page?.pageOverview?.elements ?? []).some((el) => el.issueId === issue.id)
	);

	// Evidence is meaningful when there's a screenshot, an overlay we can render,
	// a page-level screenshot to anchor a page-global finding, or any text/DOM data
	// on the primary occurrence. We never show an empty tab.
	const hasEvidence = $derived(
		Boolean(
			screenshotUrl ||
			pageOverviewRenderable ||
			pageOverviewExists ||
			primaryOccurrence?.selector ||
			primaryOccurrence?.ancestorPath ||
			primaryOccurrence?.contextHtml ||
			primaryOccurrence?.html ||
			primaryOccurrence?.failureSummary
		)
	);

	const tabs = $derived(
		[
			{ id: 'fix', label: 'Fix' },
			hasEvidence ? { id: 'evidence', label: 'Evidence' } : null,
			{ id: 'details', label: 'Details' },
			occurrenceCount > 0 ? { id: 'occurrences', label: `Occurrences (${occurrenceCount})` } : null
		].filter((tab): tab is { id: string; label: string } => tab !== null)
	);

	const defaultTabId = $derived(hasEvidence ? 'evidence' : 'fix');
	let activeTab = $state('fix');

	$effect(() => {
		issue.id;
		highlightedElementId;
		fullPageEvidenceOverride = null;
		activeTab = defaultTabId;
	});

	// If the active tab disappears because the new issue is shaped differently,
	// fall back to the first available tab to avoid a blank panel.
	$effect(() => {
		if (!tabs.some((t) => t.id === activeTab)) {
			activeTab = tabs[0]?.id ?? 'fix';
		}
	});

	$effect(() => {
		if (!activeHighlightId || !modalRef) return;
		let cancelled = false;
		void (async () => {
			await tick();
			if (cancelled) return;
			const node = modalRef?.querySelector<HTMLElement>(
				`[data-occurrence-id="${CSS.escape(activeHighlightId)}"]`
			);
			if (typeof node?.scrollIntoView === 'function') {
				node.scrollIntoView({ block: 'center' });
			}
		})();
		return () => {
			cancelled = true;
		};
	});

	function handleKeydown(event: KeyboardEvent) {
		const target = event.target as HTMLElement | null;
		const tag = target?.tagName;
		if (tag === 'INPUT' || tag === 'TEXTAREA' || target?.isContentEditable) return;
		if (event.metaKey || event.ctrlKey || event.altKey) return;

		if (event.key === 'j' || event.key === 'ArrowDown' || event.key === 'ArrowRight') {
			if (hasNext) {
				event.preventDefault();
				navigate(1);
			}
		} else if (event.key === 'k' || event.key === 'ArrowUp' || event.key === 'ArrowLeft') {
			if (hasPrev) {
				event.preventDefault();
				navigate(-1);
			}
		}
	}

	const pageLabel = $derived(page?.path ?? page?.url ?? null);
	const kindLabel = $derived(getIssueKindLabel(issueKind));
</script>

<svelte:window onkeydown={handleKeydown} />

<Modal
	open={true}
	{onClose}
	ariaLabel="Issue details"
	size="xl"
	contentClass={cn(
		panelVariants({ padding: 'none', rounded: '2xl' }),
		'max-h-[90vh] overflow-y-auto shadow-xl'
	)}
>
	<div bind:this={modalRef} class="contents">
		<div class={cn('border border-b p-6', getSeverityContainerClass(issue.severity))}>
			<div class="flex items-start justify-between gap-4">
				<div class="min-w-0">
					<div class="mb-2 flex flex-wrap items-center gap-2">
						<span
							class={cn(
								'inline-flex items-center gap-1.5 rounded border px-2 py-0.5 font-mono text-[11px] font-semibold tracking-wide uppercase',
								getSeverityContainerClass(issue.severity)
							)}
						>
							<span class={cn('h-1.5 w-1.5 rounded-full', getSeverityDotClass(issue.severity))}
							></span>
							{issue.severity}
						</span>
						<span
							class="border-line bg-surface text-ink-muted inline-flex items-center rounded border px-2 py-0.5 font-mono text-[11px] font-semibold tracking-wide uppercase"
						>
							{kindLabel}
						</span>
						<span class="text-ink-faint font-mono text-[11px] tracking-wide uppercase">
							{issue.scanner} · {issue.ruleId}
						</span>
					</div>
					<h2 class="text-ink text-xl leading-tight font-bold">{displayTitle}</h2>
					{#if isManual}
						<p class="text-ink-muted mt-1 text-sm">
							{issue.scanner === 'lighthouse' ? 'Lighthouse' : issue.scanner} flagged this for human verification
							— no concrete DOM target was reported.
						</p>
					{/if}
					{#if alsoDetectedBy.length}
						<p class="text-ink-muted mt-1 text-sm">
							Also detected by: {alsoDetectedBy.join(', ')}
						</p>
					{/if}
					{#if issue.friendlyNode?.label}
						<p class="text-ink-muted mt-2 text-sm">{issue.friendlyNode.label}</p>
					{/if}
				</div>
				<div class="flex shrink-0 items-center gap-1">
					{#if onNavigate && totalCount > 1}
						<button
							type="button"
							onclick={() => navigate(-1)}
							disabled={!hasPrev}
							class="text-ink-muted hover:text-ink rounded p-1 transition-colors disabled:cursor-not-allowed disabled:opacity-40"
							aria-label="Previous issue (k)"
							title="Previous (k)"
						>
							<ChevronLeft class="h-5 w-5" />
						</button>
						<span class="text-ink-faint font-mono text-xs tabular-nums">
							{currentIndex + 1} / {totalCount}
						</span>
						<button
							type="button"
							onclick={() => navigate(1)}
							disabled={!hasNext}
							class="text-ink-muted hover:text-ink rounded p-1 transition-colors disabled:cursor-not-allowed disabled:opacity-40"
							aria-label="Next issue (j)"
							title="Next (j)"
						>
							<ChevronRight class="h-5 w-5" />
						</button>
					{/if}
					<button
						data-modal-initial-focus
						onclick={onClose}
						class="text-ink-muted hover:text-ink rounded p-1 transition-colors"
						aria-label="Close modal"
					>
						<XCircle class="h-6 w-6" />
					</button>
				</div>
			</div>

			{#if pageLabel || issue.category || occurrenceCount > 0}
				<div
					class="border-line/70 bg-surface/60 mt-4 flex flex-wrap items-center gap-x-4 gap-y-1.5 rounded-lg border px-3 py-2 text-xs"
				>
					{#if pageLabel}
						<span class="inline-flex min-w-0 items-center gap-1.5">
							<Globe class="text-ink-faint h-3 w-3 shrink-0" aria-hidden="true" />
							<span class="text-ink-faint font-mono tracking-wide uppercase">Page</span>
							<span class="text-ink min-w-0 truncate font-mono">{pageLabel}</span>
						</span>
					{/if}
					{#if issue.category}
						<span class="inline-flex items-center gap-1.5">
							<span class="text-ink-faint font-mono tracking-wide uppercase">Category</span>
							<span class="text-ink">{issue.category}</span>
						</span>
					{/if}
					{#if occurrenceCount > 0}
						<span class="inline-flex items-center gap-1.5">
							<span class="text-ink-faint font-mono tracking-wide uppercase">Occurrences</span>
							<span class="text-ink font-mono">{occurrenceCount}</span>
						</span>
					{/if}
					{#if primaryOccurrence?.selector}
						<span class="inline-flex min-w-0 items-center gap-1.5">
							<span class="text-ink-faint font-mono tracking-wide uppercase">Selector</span>
							<span class="text-ink min-w-0 truncate font-mono">{primaryOccurrence.selector}</span>
						</span>
					{/if}
				</div>
			{/if}
		</div>

		<div class="px-6 pt-4">
			<Tabs {tabs} value={activeTab} onValueChange={(id) => (activeTab = id)} />
		</div>

		<div class="space-y-6 p-6">
			{#if activeTab === 'fix'}
				<div>
					<h3 class="text-ink mb-2 font-semibold">How to fix</h3>
					{#if contextualFix}
						<p class="text-ink-muted text-sm leading-relaxed whitespace-pre-wrap">
							{contextualFix}
						</p>
					{:else}
						<p class="text-ink-muted text-sm italic">
							No fix guidance available for this rule yet.
						</p>
					{/if}
				</div>
				{#if issue.helpUrl}
					<a
						href={issue.helpUrl}
						target="_blank"
						rel="noopener noreferrer"
						class="text-accent inline-flex items-center gap-2 text-sm hover:underline"
					>
						<FileText class="h-4 w-4" />
						Learn more about this issue
						<ExternalLink class="h-3 w-3" />
					</a>
				{/if}
			{:else if activeTab === 'evidence'}
				{#if openedFromOverlay && pageOverviewRenderable}
					<div
						class="border-line bg-surface-muted/40 flex items-center justify-between rounded-lg border border-dashed px-3 py-2"
					>
						<p class="text-ink-muted text-xs">Opened from page highlight.</p>
						<button
							type="button"
							class="text-accent hover:text-accent-strong text-xs font-semibold"
							onclick={() => {
								fullPageEvidenceOverride = !showFullPageEvidence;
							}}
						>
							{showFullPageEvidence ? 'Hide full page context' : 'Show full page context'}
						</button>
					</div>
				{/if}
				<IssueEvidenceSection
					{issue}
					{page}
					{screenshotUrl}
					{pageOverviewUrl}
					showPageOverview={shouldShowPageOverview}
					hideScreenshot={openedFromOverlay && shouldShowPageOverview && pageOverviewRenderable}
					onElementClick={(elementId) => {
						localHighlightedElementId = elementId;
					}}
				/>
			{:else if activeTab === 'details'}
				<div>
					<h3 class="text-ink mb-2 font-semibold">Description</h3>
					<p class="text-ink-muted text-sm leading-relaxed">{issue.description}</p>
				</div>
				{#if issue.wcagTags?.length}
					<div>
						<h3 class="text-ink mb-2 font-semibold">WCAG criteria</h3>
						<div class="flex flex-wrap gap-2">
							{#each issue.wcagTags as tag (tag)}
								{@const criterion = normalizeWcagTag(tag)}
								{#if criterion}
									<Chip
										as="a"
										href={getWcagUnderstandingUrl(criterion)}
										target="_blank"
										rel="noopener noreferrer"
										tone="muted"
										size="md"
										interactive
										title={`Open WCAG Understanding ${criterion}`}
									>
										{criterion}
									</Chip>
								{:else}
									<Chip tone="muted" size="md">{tag}</Chip>
								{/if}
							{/each}
						</div>
					</div>
				{/if}
				<div class="grid gap-3 sm:grid-cols-2">
					<div>
						<p class="text-ink-faint font-mono text-[11px] tracking-wide uppercase">Scanner</p>
						<p class="text-ink mt-0.5 font-mono text-sm">{issue.scanner}</p>
					</div>
					<div>
						<p class="text-ink-faint font-mono text-[11px] tracking-wide uppercase">Rule ID</p>
						<p class="text-ink mt-0.5 font-mono text-sm">{issue.ruleId}</p>
					</div>
					{#if issue.category}
						<div>
							<p class="text-ink-faint font-mono text-[11px] tracking-wide uppercase">Category</p>
							<p class="text-ink mt-0.5 text-sm">{issue.category}</p>
						</div>
					{/if}
					<div>
						<p class="text-ink-faint font-mono text-[11px] tracking-wide uppercase">Severity</p>
						<p class="text-ink mt-0.5 text-sm capitalize">{issue.severity}</p>
					</div>
					<div>
						<p class="text-ink-faint font-mono text-[11px] tracking-wide uppercase">Issue kind</p>
						<p class="text-ink mt-0.5 text-sm">{kindLabel}</p>
					</div>
					{#if pageLabel}
						<div class="min-w-0">
							<p class="text-ink-faint font-mono text-[11px] tracking-wide uppercase">Page</p>
							<p class="text-ink mt-0.5 truncate font-mono text-sm" title={pageLabel}>
								{pageLabel}
							</p>
						</div>
					{/if}
				</div>
			{:else if activeTab === 'occurrences'}
				<div class="space-y-3">
					<h3 class="text-ink font-semibold">
						Affected elements ({occurrenceCount})
					</h3>
					{#each issue.occurrences ?? [] as occurrence, idx (idx)}
						{@const isHighlighted = activeHighlightId && occurrence.elementId === activeHighlightId}
						<IssueOccurrenceCard
							{occurrence}
							index={idx}
							{issue}
							{page}
							{pageOverviewUrl}
							isHighlighted={!!isHighlighted}
							showDetails={true}
							hideThumbnail={false}
							onHighlight={() => {
								localHighlightedElementId = occurrence.elementId ?? null;
							}}
						/>
					{/each}
				</div>
			{/if}
		</div>
	</div>
</Modal>
