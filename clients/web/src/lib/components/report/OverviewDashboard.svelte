<script lang="ts">
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { Panel, chipVariants } from '$lib/components/ui';
	import {
		formatScannerStatus,
		getScannerStatusTone,
		getSeverityDotClass,
		severityRank
	} from '$lib/report';
	import { cn } from '$lib/utils';
	import { AlertTriangle, ArrowRight, CheckCircle2, MinusCircle, XCircle } from 'lucide-svelte';

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

	const highSeverityCount = $derived(
		(report.summary.bySeverity?.critical ?? 0) + (report.summary.bySeverity?.serious ?? 0)
	);

	// Distinct findings (one per scanner+rule), so the headline reflects real
	// problems rather than the per-page occurrence count. The same console error
	// seen on 8 pages is one issue, not eight.
	const distinctIssueCount = $derived.by(() => {
		const keys = new Set<string>();
		for (const issue of report.issues) keys.add(`${issue.scanner}::${issue.ruleId}`);
		return keys.size;
	});

	// Plain, non-alarmist headline. We report what was found, we don't rank dread.
	const riskLabel = $derived.by(() => {
		const n = distinctIssueCount;
		if (n === 0) return 'No issues found';
		return `${n.toLocaleString()} issue${n !== 1 ? 's' : ''} found`;
	});

	const riskTone = $derived(distinctIssueCount === 0 ? 'success' : 'info');

	const topRules = $derived.by(() => {
		const counts: Record<
			string,
			{
				count: number;
				title: string;
				scanner: string;
				ruleId: string;
				severity: string;
				pages: Set<string>;
			}
		> = {};
		for (const issue of report.issues) {
			const key = `${issue.scanner}::${issue.ruleId}`;
			const existing = counts[key];
			if (existing) {
				existing.count += 1;
				if (issue.pageId) existing.pages.add(issue.pageId);
				// Keep the highest-severity title we saw for the group
				if (severityRank(issue.severity) < severityRank(existing.severity)) {
					existing.severity = issue.severity;
				}
			} else {
				counts[key] = {
					count: 1,
					title: issue.title ?? issue.ruleId,
					scanner: issue.scanner,
					ruleId: issue.ruleId,
					severity: issue.severity,
					pages: new Set(issue.pageId ? [issue.pageId] : [])
				};
			}
		}
		return Object.entries(counts)
			.map(([key, meta]) => ({
				key,
				count: meta.count,
				title: meta.title,
				scanner: meta.scanner,
				ruleId: meta.ruleId,
				severity: meta.severity,
				pageIds: Array.from(meta.pages)
			}))
			.sort((a, b) => {
				const sevDiff = severityRank(a.severity) - severityRank(b.severity);
				if (sevDiff !== 0) return sevDiff;
				return b.count - a.count;
			})
			.slice(0, 5);
	});

	const topFixes = $derived(topRules.slice(0, 5));

	function pageLabelForId(pageId: string): string {
		const found = report.pages.find((p) => p.id === pageId);
		return found?.path ?? found?.url ?? pageId;
	}

	const riskSummary = $derived.by(() => {
		const distinct = distinctIssueCount;
		const occurrences = report.summary.totalIssues;
		const pagesWith = report.summary.pagesWithIssues;
		const scanned = report.summary.pagesScanned;
		if (distinct === 0) {
			if ((report.errors?.length ?? 0) > 0)
				return 'No issues detected, but some scans errored — review before treating this as clean.';
			return 'No issues detected.';
		}
		const occNote =
			occurrences !== distinct ? ` (${occurrences.toLocaleString()} occurrences)` : '';
		return `${distinct.toLocaleString()} distinct issue${distinct !== 1 ? 's' : ''}${occNote} seen across ${pagesWith.toLocaleString()} of ${scanned.toLocaleString()} page${scanned !== 1 ? 's' : ''} scanned.`;
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
	<Panel class="ring-line/70 shadow-sm ring-1" padding="md" rounded="2xl">
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
	{@const hasError = scanner.status === 'failed' || !!scanner.error}
	<div
		class={cn(
			'border-line flex flex-col gap-2 rounded-xl border p-3 transition',
			hasError ? 'border-red-200/70 bg-red-50/40' : 'hover:border-accent/50 hover:bg-surface-muted'
		)}
	>
		<button
			onclick={() => onSelectScanner(scanner.id)}
			class="-m-1 flex w-full items-center justify-between gap-3 rounded-lg p-1 text-left transition"
		>
			<div class="flex min-w-0 items-start gap-3">
				<StatusIcon
					class={cn(
						'mt-0.5 h-5 w-5 shrink-0',
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
		{#if hasError}
			<div
				class="overflow-hidden rounded-lg border border-red-200 bg-red-50/80 p-2.5 text-xs text-red-800"
			>
				<p class="text-[10px] font-semibold tracking-wider text-red-600 uppercase">
					Scanner failure
				</p>
				<p class="mt-1 font-mono break-all">
					{scanner.error || 'Terminated without producing results.'}
				</p>
			</div>
		{/if}
	</div>
{/snippet}

<div class="space-y-6">
	<Panel class="ring-line/70 shadow-sm ring-1" padding="md" rounded="2xl">
		<div class="grid gap-4 lg:grid-cols-[1.1fr,0.9fr]">
			<div class="border-line bg-surface-muted/40 rounded-2xl border p-4">
				<p class="text-ink-faint text-[11px] font-semibold tracking-[0.14em] uppercase">Summary</p>
				<p class="text-ink mt-2 text-sm font-semibold">{riskSummary}</p>
				<div class="mt-4 flex flex-wrap gap-2">
					<button
						onclick={() => onSearchIssues('', undefined)}
						class="bg-accent hover:bg-accent/90 rounded-xl px-3 py-2 text-sm font-semibold text-white transition"
					>
						Open issue list
					</button>
					{#if topPage}
						<button
							onclick={() => onSelectPage(topPage.id)}
							class="border-line bg-surface text-ink hover:bg-surface-muted rounded-xl border px-3 py-2 text-sm font-semibold transition"
						>
							Inspect most-affected page
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
		padding="lg"
		rounded="2xl"
	>
		<div
			class="bg-accent/12 pointer-events-none absolute -top-18 -right-14 h-44 w-44 rounded-full blur-3xl"
		></div>
		<div class="relative flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
			<div>
				<p
					class="text-ink-faint flex items-center text-xs font-semibold tracking-[0.12em] uppercase"
				>
					<span
						class="text-accent bg-accent-soft mr-2 rounded px-1.5 py-0.5 font-mono text-[9px] leading-none font-bold"
						>01</span
					>
					Overview
				</p>
				<div class="mt-2 flex flex-wrap items-center gap-2">
					<p class="text-ink text-2xl font-bold sm:text-3xl">{riskLabel}</p>
					<span
						class={cn(
							'inline-flex items-center rounded-full border px-2.5 py-1 text-xs font-semibold tracking-wide',
							getRiskChipClass(riskTone)
						)}
					>
						{report.summary.pagesWithIssues.toLocaleString()} of {report.summary.pagesScanned.toLocaleString()}
						pages affected
					</span>
				</div>
				<p class="text-ink-muted mt-2 max-w-2xl text-sm">{riskSummary}</p>
			</div>
			<div class="grid grid-cols-3 gap-3 text-right lg:min-w-[380px]">
				<div>
					<p class="text-ink-faint text-[11px] font-semibold tracking-[0.1em] uppercase">
						Pages affected
					</p>
					<p class="text-ink mt-1 text-2xl font-bold">{report.summary.pagesWithIssues}</p>
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
		{@render summaryCard('Distinct Issues', distinctIssueCount)}
		{@render summaryCard('Occurrences', report.summary.totalIssues)}
		{@render summaryCard('Pages Scanned', report.summary.pagesScanned)}
		{@render summaryCard('Pages With Issues', report.summary.pagesWithIssues, pagesWithIssuesTone)}
	</div>

	{#if topFixes.length > 0}
		<Panel class="ring-line/70 shadow-sm ring-1" padding="none" rounded="2xl">
			<div class="border-line flex items-center justify-between gap-3 border-b p-4">
				<h3 class="text-ink flex items-center text-base leading-none font-semibold tracking-tight">
					<span
						class="text-accent bg-accent-soft mr-2 rounded px-1.5 py-0.5 font-mono text-[9px] leading-none font-bold"
						>FIX</span
					>
					Top fixes — start here
				</h3>
				<button
					type="button"
					onclick={() => onSearchIssues('', undefined)}
					class="text-accent hover:text-accent-strong inline-flex items-center gap-1 text-xs font-semibold"
				>
					Open issue list
					<ArrowRight class="h-3 w-3" aria-hidden="true" />
				</button>
			</div>
			<ol class="divide-line/70 divide-y" data-testid="top-fixes">
				{#each topFixes as fix, idx (fix.key)}
					{@const previewPages = fix.pageIds.slice(0, 3)}
					{@const remainingPages = Math.max(0, fix.pageIds.length - previewPages.length)}
					<li>
						<button
							type="button"
							onclick={() => onSearchIssues(fix.ruleId, fix.scanner)}
							class="hover:bg-surface-muted/60 flex w-full items-start gap-4 px-4 py-3 text-left transition"
						>
							<span
								class="text-ink-faint mt-0.5 w-6 shrink-0 font-mono text-sm font-semibold tabular-nums"
								>#{idx + 1}</span
							>
							<div class="min-w-0 flex-1">
								<div class="flex flex-wrap items-center gap-2">
									<span
										class={cn(
											'inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-[10px] leading-none font-semibold tracking-wide uppercase',
											'bg-surface'
										)}
									>
										<span class={cn('h-1.5 w-1.5 rounded-full', getSeverityDotClass(fix.severity))}
										></span>
										{fix.severity}
									</span>
									<span class="text-ink-faint font-mono text-[10px] tracking-wide uppercase">
										{fix.scanner} · {fix.ruleId}
									</span>
								</div>
								<p class="text-ink mt-1.5 text-sm font-semibold">{fix.title}</p>
								<div
									class="text-ink-muted mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs"
								>
									<span class="font-mono">
										{fix.count.toLocaleString()}
										{fix.count === 1 ? 'occurrence' : 'occurrences'} · {fix.pageIds.length}
										{fix.pageIds.length === 1 ? 'page' : 'pages'}
									</span>
									{#if previewPages.length}
										<span class="text-ink-faint inline-flex flex-wrap gap-1.5">
											{#each previewPages as pageId (pageId)}
												<span
													class="border-line/60 bg-surface-muted text-ink-muted truncate rounded border px-1.5 py-0.5 font-mono text-[10px]"
													title={pageLabelForId(pageId)}
												>
													{pageLabelForId(pageId)}
												</span>
											{/each}
											{#if remainingPages > 0}
												<span class="text-ink-faint text-[10px]">+{remainingPages} more</span>
											{/if}
										</span>
									{/if}
								</div>
							</div>
							<ArrowRight class="text-ink-faint mt-1 h-4 w-4 shrink-0" aria-hidden="true" />
						</button>
					</li>
				{/each}
			</ol>
		</Panel>
	{/if}

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

	<div class="grid grid-cols-1 gap-6 lg:grid-cols-[1.4fr,0.86fr]">
		<Panel class="ring-line/70 min-w-0 shadow-sm ring-1" padding="none" rounded="2xl">
			<div class="border-line border-b p-4">
				<h3 class="text-ink flex items-center text-base leading-none font-semibold tracking-tight">
					<span
						class="text-accent bg-accent-soft mr-2 rounded px-1.5 py-0.5 font-mono text-[9px] leading-none font-bold"
						>02</span
					>
					Scanner Status
				</h3>
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
					<h3
						class="text-ink flex items-center text-base leading-none font-semibold tracking-tight"
					>
						<span
							class="text-accent bg-accent-soft mr-2 rounded px-1.5 py-0.5 font-mono text-[9px] leading-none font-bold"
							>03</span
						>
						Top Pages
					</h3>
				</div>
				{#each topPages as page (page.id)}
					<button
						onclick={() => onSelectPage(page.id)}
						class="border-line hover:bg-surface-muted flex w-full items-center justify-between border-b px-4 py-3 text-left text-sm transition last:border-b-0"
					>
						<span class="text-ink truncate font-medium">
							{page.path ?? page.url}
						</span>
						<span class="text-ink-muted font-mono text-xs">{page.issueCount} issues</span>
					</button>
				{/each}
			</Panel>

			<Panel class="ring-line/70 shadow-sm ring-1" padding="none" rounded="2xl">
				<div class="border-line border-b p-4">
					<h3
						class="text-ink flex items-center text-base leading-none font-semibold tracking-tight"
					>
						<span
							class="text-accent bg-accent-soft mr-2 rounded px-1.5 py-0.5 font-mono text-[9px] leading-none font-bold"
							>04</span
						>
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
							<span class="text-ink-muted font-mono text-xs">
								{rule.scanner} · {rule.count} occurrences
							</span>
						</button>
					{/each}
				</div>
			</Panel>

			<Panel class="ring-line/70 shadow-sm ring-1" padding="none" rounded="2xl">
				<div class="border-line border-b p-4">
					<h3
						class="text-ink flex items-center text-base leading-none font-semibold tracking-tight"
					>
						<span
							class="text-accent bg-accent-soft mr-2 rounded px-1.5 py-0.5 font-mono text-[9px] leading-none font-bold"
							>05</span
						>
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
							<span>Distinct issues</span>
							<span>{distinctIssueCount.toLocaleString()}</span>
						</div>
						<p class="text-ink-faint text-xs leading-relaxed">
							{report.summary.totalIssues.toLocaleString()} total occurrence{report.summary
								.totalIssues !== 1
								? 's'
								: ''} across {report.scanners.length} scanner{report.scanners.length !== 1
								? 's'
								: ''}.
						</p>
					</div>
				</div>
			</Panel>
		</div>
	</div>
</div>
