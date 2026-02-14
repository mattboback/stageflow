<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { chipVariants, Panel } from '$lib/components/ui';
	import {
		formatScannerStatus,
		getScannerStatusTone,
		summarizeIssuesByPage,
		summarizeIssuesByRule
	} from '$lib/report';
	import { cn, formatDuration } from '$lib/utils';
	import { ExternalLink } from 'lucide-svelte';

	import LighthouseSummary from './LighthouseSummary.svelte';
	import SeverityBreakdown from './SeverityBreakdown.svelte';

	interface Props {
		report: UnifiedReport;
		job: ScanResult | null;
		activeScanner: string | null;
		onSelectScanner: (scannerId: string) => void;
	}

	const { report, job, activeScanner, onSelectScanner }: Props = $props();

	const selectedScanner = $derived(
		report.scanners.find((scanner) => scanner.id === activeScanner) ?? null
	);

	const scannerArtifacts = $derived(job?.artifacts?.scanner_artifacts ?? {});

	const scannerIssues = $derived.by(() => {
		if (!selectedScanner) return [];
		return report.issues.filter((issue) => issue.scanner === selectedScanner.id);
	});

	const pagesById = $derived.by(() =>
		Object.fromEntries(report.pages.map((page) => [page.id, page]))
	);
	const issuesByPage = $derived.by(() => summarizeIssuesByPage(scannerIssues, pagesById));
	const issuesByRule = $derived.by(() => summarizeIssuesByRule(scannerIssues));

	const scannerDetailSections = $derived.by(() => [
		{ id: 'security-headers', title: 'Header gaps', items: issuesByRule.slice(0, 10).map(i => ({ key: i.ruleId, label: i.ruleId, count: i.count })) },
		{ id: 'seo', title: 'SEO topics', items: issuesByRule.slice(0, 10).map(i => ({ key: i.ruleId, label: i.ruleId, count: i.count })) },
		{ id: 'link-checker', title: 'Broken links by page', items: issuesByPage.slice(0, 12).map(i => ({ key: i.pageId, label: i.label, count: i.count })) }
	]);
</script>

{#snippet scannerCard(scanner: UnifiedReport['scanners'][number])}
	{@const statusTone = getScannerStatusTone(scanner.status)}
	{@const issueCount = scanner.issueCount ?? report.summary.byScanner?.[scanner.id] ?? 0}
	<button
		onclick={() => onSelectScanner(scanner.id)}
		class={cn(
			'border-line hover:border-accent/50 rounded-2xl border p-4 text-left transition',
			activeScanner === scanner.id && 'bg-accent/5 border-accent'
		)}
	>
		<div class="flex items-start justify-between gap-3">
			<div class="min-w-0">
				<p class="text-ink truncate font-semibold">{scanner.name ?? scanner.id}</p>
				<p class="text-ink-muted text-xs">{issueCount} issues</p>
			</div>
			<span class={cn(chipVariants({ tone: statusTone, size: 'xs', caps: true }))}>
				{formatScannerStatus(scanner.status)}
			</span>
		</div>
		{#if scanner.error}
			<p class="mt-2 text-xs text-red-600">{scanner.error}</p>
		{/if}
		{#if scanner.durationMs}
			<p class="text-ink-faint mt-2 text-xs">{formatDuration(scanner.durationMs)}</p>
		{/if}
	</button>
{/snippet}

{#snippet statRow(label: string, count: number | string)}
	<div class="flex items-center justify-between gap-3">
		<span class="text-ink-muted truncate">{label}</span>
		<span class="text-ink text-xs font-semibold">{count}</span>
	</div>
{/snippet}

<div class="space-y-6">
	<Panel class="shadow-sm" padding="none" rounded="2xl">
		<div class="border-line border-b p-4">
			<h3 class="text-ink text-base leading-none font-semibold tracking-tight">Scanner Results</h3>
		</div>
		<div class="p-4">
			<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
				{#each report.scanners as scanner (scanner.id)}
					{@render scannerCard(scanner)}
				{/each}
			</div>
		</div>
	</Panel>

	{#if selectedScanner}
		{@const artifacts = scannerArtifacts[selectedScanner.id]}
		{@const selectedStatusTone = getScannerStatusTone(selectedScanner.status)}
		{@const issueCount = selectedScanner.issueCount ?? scannerIssues.length}
		<Panel class="shadow-sm" padding="none" rounded="2xl">
			<div class="border-line border-b p-4">
				<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
					{selectedScanner.name ?? selectedScanner.id} details
				</h3>
			</div>
			<div class="space-y-6 p-4">
				<div class="flex flex-wrap items-center gap-3 text-sm">
					<span class={cn(chipVariants({ tone: selectedStatusTone, size: 'xs', caps: true }))}>
						{formatScannerStatus(selectedScanner.status)}
					</span>
					<span class="text-ink-muted">{issueCount} issues</span>
					{#if selectedScanner.toolVersion}
						<span class="text-ink-muted">Version: {selectedScanner.toolVersion}</span>
					{/if}
					{#if selectedScanner.durationMs}
						<span class="text-ink-muted"
							>Duration: {formatDuration(selectedScanner.durationMs)}</span
						>
					{/if}
				</div>
				{#if selectedScanner.error}
					<div class="rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-700">
						{selectedScanner.error}
					</div>
				{/if}
				{#if selectedScanner.severity}
					<div>
						<p class="text-ink mb-3 text-sm font-semibold">Severity breakdown</p>
						<SeverityBreakdown bySeverity={selectedScanner.severity} />
					</div>
				{/if}
				{#if artifacts?.results_json || artifacts?.report_html}
					<div class="flex flex-wrap gap-3 text-sm">
						{#if artifacts?.results_json}
							<a
								href={artifacts.results_json}
								target="_blank"
								rel="noopener noreferrer"
								class="text-accent inline-flex items-center gap-2 hover:underline"
							>
								JSON results
								<ExternalLink class="h-3 w-3" />
							</a>
						{/if}
						{#if artifacts?.report_html}
							<a
								href={artifacts.report_html}
								target="_blank"
								rel="noopener noreferrer"
								class="text-accent inline-flex items-center gap-2 hover:underline"
							>
								HTML report
								<ExternalLink class="h-3 w-3" />
							</a>
						{/if}
					</div>
				{:else}
					<p class="text-ink-muted text-sm">No artifacts published for this scanner yet.</p>
				{/if}

				{#if selectedScanner.id === 'lighthouse' && report.summary.lighthouseCategories?.length}
					<div>
						<p class="text-ink mb-3 text-sm font-semibold">Lighthouse categories</p>
						<LighthouseSummary categories={report.summary.lighthouseCategories} />
					</div>
				{/if}

				{#if scannerIssues.length > 0}
					<div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
						<Panel padding="sm" rounded="xl">
							<p class="text-ink mb-2 text-sm font-semibold">Top findings</p>
							<div class="space-y-2 text-sm">
								{#each issuesByRule.slice(0, 8) as item (item.ruleId)}
									{@render statRow(item.title, item.count)}
								{/each}
							</div>
						</Panel>
						<Panel padding="sm" rounded="xl">
							<p class="text-ink mb-2 text-sm font-semibold">Findings by page</p>
							<div class="space-y-2 text-sm">
								{#each issuesByPage.slice(0, 8) as item (item.pageId)}
									{@render statRow(item.label, item.count)}
								{/each}
							</div>
						</Panel>
					</div>
				{:else}
					<Panel variant="muted" padding="sm" rounded="xl">
						<p class="text-ink-muted text-sm">No issues reported by this scanner.</p>
					</Panel>
				{/if}

				{#each scannerDetailSections as detail (detail.id)}
					{#if selectedScanner.id === detail.id && detail.items.length > 0}
						<Panel padding="sm" rounded="xl">
							<p class="text-ink mb-2 text-sm font-semibold">{detail.title}</p>
							<div class="space-y-2 text-sm">
								{#each detail.items as item (item.key)}
									{@render statRow(item.label, item.count)}
								{/each}
							</div>
						</Panel>
					{/if}
				{/each}
			</div>
		</Panel>
	{:else}
		<Panel variant="muted" padding="lg" rounded="2xl" class="text-ink-muted text-center text-sm">
			Select a scanner to see details.
		</Panel>
	{/if}
</div>
