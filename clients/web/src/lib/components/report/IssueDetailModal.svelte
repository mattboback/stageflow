<script lang="ts">
	import type { ReportAudience } from '$lib/types/report-audience';
	import type { ScreenshotArtifact } from '$lib/types/scan';
	import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

	import { Chip, Modal, panelVariants } from '$lib/components/ui';
	import {
		getIssueScreenshotUrl,
		getPageOverviewUrl,
		getSeverityBadgeClass,
		getSeverityBorderClass
	} from '$lib/report';
	import { cn, getWcagUnderstandingUrl, normalizeWcagTag } from '$lib/utils';
	import { extractPrimaryFailureDetail } from '$lib/utils/failure-summary';
	import { ExternalLink, FileText, XCircle } from 'lucide-svelte';
	import { tick } from 'svelte';

	import IssueEvidenceSection from './IssueEvidenceSection.svelte';
	import IssueOccurrenceCard from './IssueOccurrenceCard.svelte';
	import IssueTriageCard from './IssueTriageCard.svelte';

	interface Props {
		issue: IssueDetail;
		page?: PageSummary | null;
		audience?: ReportAudience;
		screenshots: ScreenshotArtifact[];
		highlightedElementId?: string;
		onClose: () => void;
	}

	let {
		issue,
		page = null,
		audience = 'pm',
		screenshots,
		highlightedElementId,
		onClose
	}: Props = $props();

	const isEngineer = $derived(audience === 'engineer');
	const isPM = $derived(audience === 'pm');

	const primaryOccurrence = $derived(issue.occurrences?.[0] ?? null);
	const primaryFix = $derived.by(() => {
		if (issue.howToFix) {
			const trimmed = issue.howToFix.trim();
			const firstLine = trimmed
				.split('\n')
				.map((line) => line.trim())
				.find(Boolean);
			if (!firstLine) return null;
			return firstLine.length > 220 ? `${firstLine.slice(0, 217)}...` : firstLine;
		}
		if (primaryOccurrence?.failureSummary) {
			const detail =
				extractPrimaryFailureDetail(primaryOccurrence.failureSummary) ??
				primaryOccurrence.failureSummary;
			const trimmed = detail.trim();
			if (!trimmed) return null;
			return trimmed.length > 220 ? `${trimmed.slice(0, 217)}...` : trimmed;
		}
		return null;
	});

	const suggestedOwner = $derived.by(() => {
		const ruleId = (issue.ruleId ?? '').toLowerCase();
		const category = (issue.category ?? '').toLowerCase();
		const tags = (issue.wcagTags ?? []).map((t) => t.toLowerCase());

		if (
			ruleId.includes('image-alt') ||
			ruleId.includes('document-title') ||
			ruleId.includes('meta-description') ||
			category.includes('seo')
		) {
			return 'Content';
		}

		if (
			ruleId.includes('color-contrast') ||
			ruleId.includes('contrast') ||
			ruleId.includes('focus') ||
			tags.some((t) => t.includes('1.4.3') || t.includes('1.4.11'))
		) {
			return 'Design';
		}

		return 'Engineering';
	});

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
	const pageOverviewRenderable = $derived(
		!!page &&
			!!pageOverviewUrl &&
			(page.pageOverview?.pageWidth ?? 0) > 0 &&
			(page.pageOverview?.pageHeight ?? 0) > 0
	);

	$effect(() => {
		issue.id;
		highlightedElementId;
		fullPageEvidenceOverride = null;
	});

	// Scroll to highlighted element when it changes
	$effect(() => {
		if (!activeHighlightId || !modalRef) return;

		let cancelled = false;

		void (async () => {
			await tick();
			// Bail if effect re-ran before tick completed
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
</script>

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
		<div class={cn('border-b p-6', getSeverityBorderClass(issue.severity))}>
			<div class="flex items-start justify-between gap-4">
				<div>
					<span
						class={cn(
							'mb-2 inline-block rounded px-2 py-0.5 text-xs font-semibold uppercase',
							getSeverityBadgeClass(issue.severity)
						)}
					>
						{issue.severity}
					</span>
					<h2 class="text-ink text-xl font-bold">{issue.title}</h2>
					<p class="text-ink-muted mt-1 text-sm">
						{issue.scanner} &middot; Rule: {issue.ruleId}
					</p>
					{#if alsoDetectedBy.length}
						<p class="text-ink-muted mt-1 text-sm">Also detected by: {alsoDetectedBy.join(', ')}</p>
					{/if}
					{#if issue.friendlyNode?.label}
						<p class="text-ink-muted mt-2 text-sm">{issue.friendlyNode.label}</p>
					{/if}
				</div>
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

		<div class="space-y-6 p-6">
			<!-- PM Triage Card (PM view only) -->
			{#if isPM}
				<IssueTriageCard {issue} {page} {pageOverviewUrl} {suggestedOwner} {primaryFix} />
			{/if}
			{#snippet descriptionBlock()}
				<div>
					<h3 class="text-ink mb-2 font-semibold">Description</h3>
					<p class="text-ink-muted text-sm leading-relaxed">{issue.description}</p>
				</div>
			{/snippet}

			{#snippet wcagBlock()}
				{#if issue.wcagTags?.length}
					<div>
						<h3 class="text-ink mb-2 font-semibold">WCAG Criteria</h3>
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
									<Chip tone="muted" size="md">
										{tag}
									</Chip>
								{/if}
							{/each}
						</div>
					</div>
				{/if}
			{/snippet}

			{#snippet howToFixBlock()}
				{#if issue.howToFix}
					<div>
						<h3 class="text-ink mb-2 font-semibold">How to Fix</h3>
						<p class="text-ink-muted text-sm leading-relaxed whitespace-pre-wrap">
							{issue.howToFix}
						</p>
					</div>
				{/if}
			{/snippet}

			{#snippet evidenceBlock()}
				{#if screenshotUrl || pageOverviewUrl}
					{#if openedFromOverlay}
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
				{/if}
			{/snippet}

			{#snippet occurrencesBlock(limit: number, showDetails: boolean)}
				{#if issue.occurrences?.length}
					<div>
						<h3 class="text-ink mb-2 font-semibold">
							Affected Elements ({issue.occurrences.length})
						</h3>
						<div class="space-y-3">
							{#each issue.occurrences.slice(0, limit) as occurrence, idx (idx)}
								{@const isHighlighted =
									activeHighlightId && occurrence.elementId === activeHighlightId}
								<IssueOccurrenceCard
									{occurrence}
									index={idx}
									{issue}
									{page}
									{pageOverviewUrl}
									isHighlighted={!!isHighlighted}
									{showDetails}
									onHighlight={() => {
										localHighlightedElementId = occurrence.elementId ?? null;
									}}
								/>
							{/each}
							{#if issue.occurrences.length > limit}
								<p class="text-ink-muted text-center text-sm">
									+{issue.occurrences.length - limit} more occurrences
								</p>
							{/if}
						</div>
					</div>
				{/if}
			{/snippet}

			<!-- Shared Sections -->
			{#if isPM}
				<details class="border-line rounded-xl border p-4" open>
					<summary class="text-ink cursor-pointer font-semibold">Implementation details</summary>
					<div class="mt-4 space-y-6">
						{@render descriptionBlock()}
						{@render wcagBlock()}
						{@render howToFixBlock()}

						{#if screenshotUrl || pageOverviewUrl}
							<details class="border-line rounded-lg border p-3">
								<summary class="text-ink cursor-pointer text-sm font-semibold">
									Technical evidence (optional)
								</summary>
								<div class="mt-4">
									{@render evidenceBlock()}
								</div>
							</details>
						{/if}

						{@render occurrencesBlock(10, true)}
					</div>
				</details>
			{:else}
				{@render descriptionBlock()}
				{@render wcagBlock()}

				<!-- User Impact (designer view) -->
				{#if issue.userImpact?.statement && audience === 'designer'}
					<div>
						<h3 class="text-ink mb-2 font-semibold">User Impact</h3>
						<p class="text-ink-muted text-sm leading-relaxed">{issue.userImpact.statement}</p>
						{#if issue.userImpact.affectedGroups?.length}
							<div class="mt-2 flex flex-wrap gap-2">
								{#each issue.userImpact.affectedGroups as group (group)}
									<span
										class="rounded bg-purple-100 px-2 py-0.5 text-xs font-medium text-purple-700 capitalize"
									>
										{group}
									</span>
								{/each}
							</div>
						{/if}
					</div>
				{/if}

				{#if isEngineer}
					{@render howToFixBlock()}
				{/if}
				{@render evidenceBlock()}
				{#if isEngineer}
					{@render occurrencesBlock(10, true)}
				{/if}
			{/if}

			<!-- Help URL (always shown) -->
			{#if issue.helpUrl}
				<div class="border-line border-t pt-4">
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
				</div>
			{/if}
		</div>
	</div>
</Modal>
