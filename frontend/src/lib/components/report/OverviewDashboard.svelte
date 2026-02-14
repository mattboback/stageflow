<script lang="ts">
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { chipVariants, Panel } from '$lib/components/ui';
	import { formatScannerStatus, getScannerStatusTone } from '$lib/report';
	import { cn } from '$lib/utils';
	import { AlertTriangle, CheckCircle2, MinusCircle, XCircle } from 'lucide-svelte';

	import LighthouseSummary from './LighthouseSummary.svelte';
	import SeverityBreakdown from './SeverityBreakdown.svelte';

	interface Props {
		report: UnifiedReport;
		onSelectPage: (pageId: string) => void;
		onSelectScanner: (scannerId: string) => void;
		onSearchIssues: (query: string, scannerId?: string) => void;
	}

	const { report, onSelectPage, onSelectScanner, onSearchIssues }: Props = $props();

	const topPages = $derived(
		[...report.pages].sort((a, b) => b.issueCount - a.issueCount).slice(0, 5)
	);

	const topRules = $derived.by(() => {
		const counts: Record<
			string,
			{ count: number; title: string; scanner: string; ruleId: string }
		> = {};
		for (const issue of report.issues) {
			const key = `${issue.scanner}::${issue.ruleId}`;
			const existing = counts[key];
			if (existing) {
				existing.count += 1;
			} else {
				counts[key] = {
					count: 1,
					title: issue.title ?? issue.ruleId,
					scanner: issue.scanner,
					ruleId: issue.ruleId
				};
			}
		}
		return Object.entries(counts)
			.map(([key, meta]) => ({
				key,
				count: meta.count,
				title: meta.title,
				scanner: meta.scanner,
				ruleId: meta.ruleId
			}))
			.sort((a, b) => b.count - a.count)
			.slice(0, 5);
	});

	const issueTone = $derived.by((): 'warn' | 'danger' | null => {
		if (report.summary.totalIssues === 0) return null;
		const critical = report.summary.bySeverity?.critical ?? 0;
		if (critical > 0) return 'danger';
		const serious = report.summary.bySeverity?.serious ?? 0;
		if (serious > 5) return 'warn';
		return null;
	});

	const pagesWithIssuesTone = $derived.by((): 'warn' | 'danger' | null => {
		if (report.summary.pagesScanned === 0) return null;
		const ratio = report.summary.pagesWithIssues / report.summary.pagesScanned;
		if (ratio > 0.75) return 'danger';
		if (ratio > 0.5) return 'warn';
		return null;
	});

	function getStatusIcon(status: string) {
		switch (status) {
			case 'success':
				return CheckCircle2;
			case 'failed':
				return XCircle;
			case 'skipped':
				return MinusCircle;
			default:
				return AlertTriangle;
		}
	}
</script>

{#snippet summaryCard(title: string, value: number | string, tone?: 'warn' | 'danger' | null)}
	<Panel class="shadow-sm" padding="sm" rounded="2xl">
		<p class="text-ink-muted text-sm">{title}</p>
		<p
			class={cn(
				'text-2xl font-bold',
				tone === 'danger' ? 'text-red-600' : tone === 'warn' ? 'text-amber-600' : 'text-ink'
			)}
		>
			{value}
		</p>
	</Panel>
{/snippet}

{#snippet scannerRow(scanner: UnifiedReport['scanners'][number])}
	{@const StatusIcon = getStatusIcon(scanner.status)}
	{@const statusTone = getScannerStatusTone(scanner.status)}
	{@const issueCount = scanner.issueCount ?? report.summary.byScanner?.[scanner.id] ?? 0}
	<button
		onclick={() => onSelectScanner(scanner.id)}
		class="border-line hover:border-accent/50 flex w-full items-center justify-between gap-3 rounded-xl border p-3 text-left transition"
	>
		<div class="flex min-w-0 items-start gap-3">
			<StatusIcon
				class={cn(
					'mt-0.5 h-5 w-5',
					scanner.status === 'success'
						? 'text-emerald-500'
						: scanner.status === 'failed'
							? 'text-red-500'
							: scanner.status === 'skipped'
								? 'text-slate-400'
								: 'text-amber-500'
				)}
			/>
			<div class="min-w-0">
				<p class="text-ink truncate font-semibold">
					{scanner.name ?? scanner.id}
				</p>
				<p class="text-ink-muted text-xs">
					{issueCount} issues
				</p>
			</div>
		</div>
		<span class={cn(chipVariants({ tone: statusTone, size: 'xs', caps: true }))}>
			{formatScannerStatus(scanner.status)}
		</span>
	</button>
{/snippet}

<div class="space-y-6">
	<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
		{@render summaryCard('Total Issues', report.summary.totalIssues, issueTone)}
		{@render summaryCard('Pages Scanned', report.summary.pagesScanned)}
		{@render summaryCard('Pages With Issues', report.summary.pagesWithIssues, pagesWithIssuesTone)}
		{@render summaryCard('Scanners Ran', report.scanners.length)}
	</div>

	<Panel class="shadow-sm" padding="none" rounded="2xl">
		<div class="border-line border-b p-4">
			<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
				Severity Breakdown
			</h3>
		</div>
		<div class="p-4">
			<SeverityBreakdown bySeverity={report.summary.bySeverity} />
		</div>
	</Panel>

	{#if report.summary.lighthouseCategories?.length}
		<Panel class="shadow-sm" padding="none" rounded="2xl">
			<div class="border-line border-b p-4">
				<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
					Lighthouse Averages
				</h3>
			</div>
			<div class="p-4">
				<LighthouseSummary categories={report.summary.lighthouseCategories} />
			</div>
		</Panel>
	{/if}

	<div class="grid grid-cols-1 gap-6 lg:grid-cols-[1.2fr,0.8fr]">
		<Panel class="shadow-sm min-w-0" padding="none" rounded="2xl">
			<div class="border-line border-b p-4">
				<h3 class="text-ink text-base leading-none font-semibold tracking-tight">Scanner Status</h3>
			</div>
			<div class="p-4">
				<div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
					{#each report.scanners as scanner (scanner.id)}
						{@render scannerRow(scanner)}
					{/each}
				</div>
			</div>
		</Panel>

		<div class="space-y-4">
			<Panel class="shadow-sm" padding="none" rounded="2xl">
				<div class="border-line border-b p-4">
					<h3 class="text-ink text-base leading-none font-semibold tracking-tight">Top Pages</h3>
				</div>
				{#each topPages as page (page.id)}
					<button
						onclick={() => onSelectPage(page.id)}
						class="border-line hover:bg-surface-muted flex w-full items-center justify-between border-b px-4 py-3 text-left text-sm transition"
					>
						<span class="text-ink truncate font-medium">
							{page.path ?? page.url}
						</span>
						<span class="text-ink-muted text-xs">{page.issueCount} issues</span>
					</button>
				{/each}
			</Panel>

			<Panel class="shadow-sm" padding="none" rounded="2xl">
				<div class="border-line border-b p-4">
					<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
						Top Issue Rules
					</h3>
				</div>
				<div class="divide-line divide-y">
					{#each topRules as rule (rule.key)}
						<button
							onclick={() => onSearchIssues(rule.ruleId, rule.scanner)}
							class="hover:bg-surface-muted flex w-full flex-col gap-1 px-4 py-3 text-left text-sm transition"
						>
							<span class="text-ink font-medium">{rule.title}</span>
							<span class="text-ink-muted text-xs">
								{rule.scanner} · {rule.count} occurrences
							</span>
						</button>
					{/each}
				</div>
			</Panel>
		</div>
	</div>
</div>
