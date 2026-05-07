<script lang="ts">
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { Panel, chipVariants } from '$lib/components/ui';
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

	let { report, onSelectPage, onSelectScanner, onSearchIssues }: Props = $props();

	const topPages = $derived(
		[...report.pages].sort((a, b) => b.issueCount - a.issueCount).slice(0, 5)
	);

	const issueDensity = $derived.by(() => {
		if (report.summary.pagesScanned <= 0) return 0;
		return report.summary.totalIssues / report.summary.pagesScanned;
	});

	const affectedRatio = $derived.by(() => {
		if (report.summary.pagesScanned <= 0) return 0;
		return report.summary.pagesWithIssues / report.summary.pagesScanned;
	});

	const criticalRatio = $derived.by(() => {
		if (report.summary.totalIssues <= 0) return 0;
		return (report.summary.bySeverity?.critical ?? 0) / report.summary.totalIssues;
	});

	const highSeverityCount = $derived(
		(report.summary.bySeverity?.critical ?? 0) + (report.summary.bySeverity?.serious ?? 0)
	);

	const riskLabel = $derived.by(() => {
		const critical = report.summary.bySeverity?.critical ?? 0;
		const serious = report.summary.bySeverity?.serious ?? 0;
		// "High risk" only when there are critical issues OR many serious issues
		if (critical > 0 || serious >= 3) return 'High risk';
		if (serious > 0) return 'Elevated risk';
		if (report.summary.totalIssues > 0) return 'Moderate risk';
		return 'Low risk';
	});

	const riskTone = $derived.by(() => {
		if (riskLabel === 'High risk') return 'danger';
		if (riskLabel === 'Elevated risk') return 'warn';
		if (riskLabel === 'Moderate risk') return 'info';
		return 'success';
	});

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

	const riskSummary = $derived.by(() => {
		const critical = report.summary.bySeverity?.critical ?? 0;
		const serious = report.summary.bySeverity?.serious ?? 0;
		const base = `${report.summary.totalIssues.toLocaleString()} issue${report.summary.totalIssues !== 1 ? 's' : ''} across ${report.summary.pagesScanned.toLocaleString()} page${report.summary.pagesScanned !== 1 ? 's' : ''}.`;
		if (critical > 0 || serious > 0)
			return `${base} Prioritize critical and serious findings first.`;
		if (report.summary.totalIssues > 0)
			return `${base} All findings are moderate severity or below.`;
		if ((report.errors?.length ?? 0) > 0)
			return 'No issues detected, but scan errors need review before treating this as clean.';
		return 'No issues detected.';
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
		if (ratio <= 0.5) return null;
		if (highSeverityCount === 0) return 'warn';
		if (ratio > 0.75) return 'danger';
		if (ratio > 0.5) return 'warn';
		return null;
	});

	const coverageTone = $derived.by((): 'success' | 'warn' | 'danger' => {
		if (affectedRatio <= 0.5) return 'success';
		if (highSeverityCount === 0) return 'warn';
		if (affectedRatio > 0.75) return 'danger';
		return 'warn';
	});
	const topPage = $derived(topPages[0] ?? null);
	const topRule = $derived(topRules[0] ?? null);

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

	function getRiskChipClass(tone: 'danger' | 'warn' | 'info' | 'success'): string {
		switch (tone) {
			case 'danger':
				return 'border-red-200 bg-red-50 text-red-700';
			case 'warn':
				return 'border-amber-200 bg-amber-50 text-amber-700';
			case 'info':
				return 'border-blue-200 bg-blue-50 text-blue-700';
			default:
				return 'border-emerald-200 bg-emerald-50 text-emerald-700';
		}
	}
</script>

{#snippet summaryCard(title: string, value: number | string, tone?: 'warn' | 'danger' | null)}
	<Panel class="ring-line/70 shadow-sm ring-1" padding="sm" rounded="2xl">
		<div class="flex min-h-24 flex-col justify-between gap-2">
			<p class="text-ink-faint text-xs font-semibold tracking-wide uppercase">{title}</p>
			<p
				class={cn(
					'text-3xl leading-none font-bold',
					tone === 'danger' ? 'text-red-600' : tone === 'warn' ? 'text-amber-600' : 'text-ink'
				)}
			>
				{value}
			</p>
		</div>
	</Panel>
{/snippet}

{#snippet scannerRow(scanner: UnifiedReport['scanners'][number])}
	{@const StatusIcon = getStatusIcon(scanner.status)}
	{@const statusTone = getScannerStatusTone(scanner.status)}
	{@const issueCount = scanner.issueCount ?? report.summary.byScanner?.[scanner.id] ?? 0}
	<button
		onclick={() => onSelectScanner(scanner.id)}
		class="border-line hover:border-accent/50 hover:bg-surface-muted flex w-full items-center justify-between gap-3 rounded-xl border p-3 text-left transition"
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
	<Panel class="ring-line/70 shadow-sm ring-1" padding="md" rounded="2xl">
		<div class="grid gap-4 lg:grid-cols-[1.1fr,0.9fr]">
			<div class="rounded-2xl border border-red-100 bg-red-50/70 p-4">
				<p class="text-[11px] font-semibold tracking-[0.14em] text-red-700 uppercase">
					Priority path
				</p>
				<p class="mt-2 text-sm font-semibold text-red-900">{riskSummary}</p>
				<div class="mt-4 flex flex-wrap gap-2">
					<button
						onclick={() => onSearchIssues('', undefined)}
						class="rounded-xl bg-red-700 px-3 py-2 text-sm font-semibold text-white transition hover:bg-red-800"
					>
						Open issue list
					</button>
					{#if topPage}
						<button
							onclick={() => onSelectPage(topPage.id)}
							class="rounded-xl border border-red-200 bg-white px-3 py-2 text-sm font-semibold text-red-900 transition hover:bg-red-50"
						>
							Inspect worst page
						</button>
					{/if}
				</div>
			</div>
			<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
				{#if topRule}
					<button
						onclick={() => onSearchIssues(topRule.ruleId, topRule.scanner)}
						class="border-line hover:border-accent/40 hover:bg-surface-muted rounded-2xl border p-4 text-left transition"
					>
						<p class="text-ink-faint text-[11px] font-semibold tracking-[0.14em] uppercase">
							Top recurring rule
						</p>
						<p class="text-ink mt-2 text-sm font-semibold">{topRule.title}</p>
						<p class="text-ink-muted mt-1 text-xs">
							{topRule.scanner} · {topRule.count} occurrences
						</p>
					</button>
				{/if}
				{#if topPage}
					<button
						onclick={() => onSelectPage(topPage.id)}
						class="border-line hover:border-accent/40 hover:bg-surface-muted rounded-2xl border p-4 text-left transition"
					>
						<p class="text-ink-faint text-[11px] font-semibold tracking-[0.14em] uppercase">
							Most impacted page
						</p>
						<p class="text-ink mt-2 truncate text-sm font-semibold">
							{topPage.path ?? topPage.url}
						</p>
						<p class="text-ink-muted mt-1 text-xs">{topPage.issueCount} issues</p>
					</button>
				{/if}
			</div>
		</div>
	</Panel>

	<Panel
		class="border-line/70 from-surface via-surface to-accent-soft/20 relative overflow-hidden border bg-gradient-to-br shadow-sm"
		padding="md"
		rounded="2xl"
	>
		<div
			class="bg-accent/12 pointer-events-none absolute -top-18 -right-14 h-44 w-44 rounded-full blur-3xl"
		></div>
		<div class="relative flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
			<div>
				<p class="text-ink-faint text-xs font-semibold tracking-[0.12em] uppercase">
					Risk snapshot
				</p>
				<div class="mt-2 flex flex-wrap items-center gap-2">
					<p class="text-ink text-2xl font-bold sm:text-3xl">{riskLabel}</p>
					<span
						class={cn(
							'inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold tracking-wide uppercase',
							getRiskChipClass(riskTone)
						)}
					>
						{Math.round(affectedRatio * 100)}% pages impacted
					</span>
				</div>
				<p class="text-ink-muted mt-2 max-w-2xl text-sm">{riskSummary}</p>
			</div>
			<div class="grid grid-cols-3 gap-3 text-right lg:min-w-[380px]">
				<div>
					<p class="text-ink-faint text-[11px] font-semibold tracking-[0.1em] uppercase">
						Critical share
					</p>
					<p class="text-ink mt-1 text-2xl font-bold">{Math.round(criticalRatio * 100)}%</p>
				</div>
				<div>
					<p class="text-ink-faint text-[11px] font-semibold tracking-[0.1em] uppercase">
						Issue density
					</p>
					<p class="text-ink mt-1 text-2xl font-bold">{issueDensity.toFixed(1)}</p>
				</div>
				<div>
					<p class="text-ink-faint text-[11px] font-semibold tracking-[0.1em] uppercase">
						Scanners
					</p>
					<p class="text-ink mt-1 text-2xl font-bold">{report.scanners.length}</p>
				</div>
			</div>
		</div>
	</Panel>

	<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
		{@render summaryCard('Total Issues', report.summary.totalIssues, issueTone)}
		{@render summaryCard('Pages Scanned', report.summary.pagesScanned)}
		{@render summaryCard('Pages With Issues', report.summary.pagesWithIssues, pagesWithIssuesTone)}
		{@render summaryCard('Issue Density', issueDensity.toFixed(1))}
	</div>

	<Panel class="ring-line/70 shadow-sm ring-1" padding="none" rounded="2xl">
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
		<Panel class="ring-line/70 shadow-sm ring-1" padding="none" rounded="2xl">
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
		<Panel class="ring-line/70 min-w-0 shadow-sm ring-1" padding="none" rounded="2xl">
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
			<Panel class="ring-line/70 shadow-sm ring-1" padding="none" rounded="2xl">
				<div class="border-line border-b p-4">
					<h3 class="text-ink text-base leading-none font-semibold tracking-tight">Top Pages</h3>
				</div>
				{#each topPages as page (page.id)}
					<button
						onclick={() => onSelectPage(page.id)}
						class="border-line hover:bg-surface-muted flex w-full items-center justify-between border-b px-4 py-3 text-left text-sm transition last:border-b-0"
					>
						<span class="text-ink truncate font-medium">
							{page.path ?? page.url}
						</span>
						<span class="text-ink-muted text-xs">{page.issueCount} issues</span>
					</button>
				{/each}
			</Panel>

			<Panel class="ring-line/70 shadow-sm ring-1" padding="none" rounded="2xl">
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

			<Panel class="ring-line/70 shadow-sm ring-1" padding="none" rounded="2xl">
				<div class="border-line border-b p-4">
					<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
						Coverage Heat
					</h3>
				</div>
				<div class="space-y-3 p-4">
					<div>
						<div class="text-ink-muted mb-1 flex items-center justify-between text-xs">
							<span>Pages with issues</span>
							<span>{Math.round(affectedRatio * 100)}%</span>
						</div>
						<div class="bg-surface-muted h-2 w-full overflow-hidden rounded-full">
							<div
								class={cn(
									'h-full transition-[width] duration-500',
									coverageTone === 'danger'
										? 'bg-red-500'
										: coverageTone === 'warn'
											? 'bg-amber-500'
											: 'bg-emerald-500'
								)}
								style={`width: ${Math.round(affectedRatio * 100)}%`}
							></div>
						</div>
					</div>
					<div>
						<div class="text-ink-muted mb-1 flex items-center justify-between text-xs">
							<span>Critical concentration</span>
							<span>{Math.round(criticalRatio * 100)}%</span>
						</div>
						<div class="bg-surface-muted h-2 w-full overflow-hidden rounded-full">
							<div
								class="h-full bg-red-500 transition-[width] duration-500"
								style={`width: ${Math.round(criticalRatio * 100)}%`}
							></div>
						</div>
					</div>
				</div>
			</Panel>
		</div>
	</div>
</div>
