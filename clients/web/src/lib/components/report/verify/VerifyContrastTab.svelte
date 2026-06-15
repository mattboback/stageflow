<script lang="ts">
	import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';
	import {
		type SampleSlot,
		getContrastData,
		getCroppedViewBox,
		getDefaultLargeText,
		isAxeIncompleteIssue
	} from '$lib/report';
	import { contrastVerdictsStore } from '$lib/stores/contrast-verdicts.svelte';
	import { describeMessageKey, formatRatio } from '$lib/utils/contrast';
	import { Info } from 'lucide-svelte';

	import ContrastResult from './ContrastResult.svelte';
	import ContrastSampler from './ContrastSampler.svelte';

	interface Props {
		issue: IssueDetail;
		page: PageSummary | null;
		pageOverviewUrl: string | null;
		jobId: string;
	}

	let { issue, page, pageOverviewUrl, jobId }: Props = $props();

	const contrastData = $derived(getContrastData(issue));
	const isIncomplete = $derived(isAxeIncompleteIssue(issue));
	const verdict = $derived(contrastVerdictsStore.getVerdict(jobId, issue.id));

	let fg = $state('');
	let bg = $state('');
	let largeText = $state(false);

	$effect(() => {
		issue.id;
		const data = getContrastData(issue);
		fg = data?.fgColor ?? '';
		bg = data?.bgColor ?? '';
		largeText = getDefaultLargeText(data) ?? false;
	});

	const overviewElement = $derived.by(() => {
		const elements = page?.pageOverview?.elements ?? [];
		return (
			elements.find((el) => el.issueId === issue.id && el.nodeIndex === 0) ??
			elements.find((el) => el.issueId === issue.id) ??
			null
		);
	});

	const cropViewBox = $derived.by(() => {
		const overview = page?.pageOverview;
		if (!overview || !overviewElement) return null;
		return getCroppedViewBox(overview.pageWidth, overview.pageHeight, overviewElement, {
			padding: 100,
			minWidth: 480,
			minHeight: 320
		});
	});

	const samplerAvailable = $derived(Boolean(pageOverviewUrl && cropViewBox && page?.pageOverview));

	const measuredNote = $derived.by(() => {
		if (!contrastData) return null;
		const parts: string[] = [];
		if (contrastData.fgColor) parts.push(`text ${contrastData.fgColor}`);
		if (contrastData.bgColor) parts.push(`on ${contrastData.bgColor}`);
		const numericRatio = Number(contrastData.contrastRatio);
		if (Number.isFinite(numericRatio) && numericRatio > 0) {
			parts.push(`· ${formatRatio(numericRatio)}:1`);
		}
		return parts.length > 0 ? parts.join(' ') : null;
	});

	function handlePick(slot: SampleSlot, hex: string) {
		if (slot === 'fg') fg = hex;
		else bg = hex;
	}

	function recordVerdict(value: 'pass' | 'fail', ratio: number | null) {
		contrastVerdictsStore.setVerdict(jobId, issue.id, { verdict: value, fg, bg, ratio });
	}
</script>

<div class="space-y-5">
	{#if isIncomplete}
		<Panel variant="muted" padding="md" rounded="lg">
			<div class="flex items-start gap-3">
				<Info class="text-ink-muted mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
				<div class="space-y-1 text-sm">
					<p class="text-ink font-semibold">Why this needs a human</p>
					<p class="text-ink-muted leading-relaxed">
						{describeMessageKey(contrastData?.messageKey)}
					</p>
					{#if measuredNote}
						<p class="text-ink-muted text-xs">
							axe measured <span class="stat-mono text-ink">{measuredNote}</span> before giving up.
						</p>
					{/if}
				</div>
			</div>
		</Panel>
	{:else if measuredNote}
		<p class="text-ink-muted text-sm">
			axe measured <span class="stat-mono text-ink">{measuredNote}</span> — sample the screenshot to double-check
			it.
		</p>
	{/if}

	{#if samplerAvailable && pageOverviewUrl && cropViewBox && page?.pageOverview}
		<ContrastSampler
			imageUrl={pageOverviewUrl}
			pageWidth={page.pageOverview.pageWidth}
			pageHeight={page.pageOverview.pageHeight}
			viewBox={cropViewBox}
			element={overviewElement}
			onPick={handlePick}
		/>
	{:else}
		<p class="border-line bg-surface-muted/40 text-ink-muted rounded-md border p-3 text-sm">
			No screenshot is available for this element — screenshots may have been disabled for this
			scan. Enter the colors manually below, or re-run the scan with screenshots enabled.
		</p>
	{/if}

	<ContrastResult
		{fg}
		{bg}
		ruleId={issue.ruleId}
		{largeText}
		{verdict}
		onFgChange={(value) => (fg = value)}
		onBgChange={(value) => (bg = value)}
		onSwap={() => {
			[fg, bg] = [bg, fg];
		}}
		onLargeTextChange={(value) => (largeText = value)}
		onRecord={recordVerdict}
		onClear={() => contrastVerdictsStore.clearVerdict(jobId, issue.id)}
	/>
</div>
