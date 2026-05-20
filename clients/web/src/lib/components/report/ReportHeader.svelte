<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { buildApiUrl } from '$lib/api/utils';
	import { Button, Panel, Score, SeverityBar } from '$lib/components/ui';
	import { scoreBandFor } from '$lib/report/score-band';
	import { cn, formatDuration, formatTimestamp } from '$lib/utils';
	import { AlertTriangle, ExternalLink, FileSearch, Layers3, RefreshCw } from 'lucide-svelte';

	interface Props {
		jobId: string;
		report: UnifiedReport;
		job: ScanResult | null;
		onJumpToIssues?: () => void;
		onJumpToPages?: () => void;
		onRefreshArtifacts?: () => void;
	}

	let {
		jobId,
		report,
		job: _job,
		onJumpToIssues,
		onJumpToPages,
		onRefreshArtifacts
	}: Props = $props();

	const jsonUrl = $derived(jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/results`) : null);
	const htmlUrl = $derived(jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/report`) : null);

	const scannedAt = $derived(formatTimestamp(report.meta.scannedAt));
	const duration = $derived(formatDuration(report.meta.durationMs));
	const pagesScanned = $derived(report.summary.pagesScanned ?? 0);
	const pagesWithIssues = $derived(report.summary.pagesWithIssues ?? 0);
	const score = $derived(report.summary.score ?? null);
	const severityCounts = $derived(
		report.summary.bySeverity ?? {
			critical: 0,
			serious: 0,
			moderate: 0,
			minor: 0,
			info: 0
		}
	);
	const criticalCount = $derived(severityCounts.critical ?? 0);
	const seriousCount = $derived(severityCounts.serious ?? 0);

	const triageHeadline = $derived.by(() => {
		if (criticalCount > 0)
			return `Start with ${criticalCount} critical issue${criticalCount === 1 ? '' : 's'}.`;
		if (seriousCount > 0)
			return `Prioritize ${seriousCount} serious issue${seriousCount === 1 ? '' : 's'} next.`;
		if (report.summary.totalIssues > 0)
			return 'Review the issue list to work through moderate findings.';
		if ((report.errors?.length ?? 0) > 0)
			return 'No issues detected, but scan errors need review before treating this as clean.';
		return 'No issues detected. Spot-check pages and artifacts to confirm release readiness.';
	});

	const band = $derived(scoreBandFor(score));
</script>

<Panel class="border-line/60 bg-surface mb-6 border shadow-xs" padding="lg" rounded="2xl">
	<div class="flex flex-col gap-4">
		<!-- Top row: title + actions -->
		<div class="flex flex-wrap items-start justify-between gap-3">
			<div class="min-w-0 flex-1">
				<p class="text-ink-faint text-[11px] font-semibold tracking-[0.12em] uppercase">
					Scan report
				</p>
				<h1
					class="text-ink mt-1 text-xl leading-tight font-bold break-all sm:text-2xl"
					data-testid="report-header-title"
				>
					{report.meta.baseUrl ?? `Scan ${jobId}`}
				</h1>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				{#if onJumpToIssues}
					<Button variant="default" size="sm" onclick={onJumpToIssues} class="gap-1.5">
						<FileSearch class="h-4 w-4" />
						Review issues
					</Button>
				{/if}
				{#if onJumpToPages}
					<Button variant="outline" size="sm" onclick={onJumpToPages} class="gap-1.5">
						<Layers3 class="h-4 w-4" />
						Check pages
					</Button>
				{/if}
				{#if jsonUrl}
					<a
						href={jsonUrl}
						target="_blank"
						rel="noopener noreferrer"
						class="border-line text-ink-muted hover:border-accent hover:text-accent inline-flex items-center gap-1 rounded-md border px-2.5 py-1.5 font-mono text-xs font-medium transition-colors"
					>
						JSON
						<ExternalLink class="h-3 w-3" />
					</a>
				{/if}
				{#if htmlUrl}
					<a
						href={htmlUrl}
						target="_blank"
						rel="noopener noreferrer"
						class="border-line text-ink-muted hover:border-accent hover:text-accent inline-flex items-center gap-1 rounded-md border px-2.5 py-1.5 font-mono text-xs font-medium transition-colors"
					>
						HTML
						<ExternalLink class="h-3 w-3" />
					</a>
				{/if}
				{#if onRefreshArtifacts}
					<Button variant="outline" size="sm" onclick={onRefreshArtifacts} class="gap-1.5">
						<RefreshCw class="h-4 w-4" />
						Refresh
					</Button>
				{/if}
			</div>
		</div>

		<!-- Condensed stats bar -->
		<div
			class="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-[auto_1fr_auto] sm:items-center"
			data-testid="report-stats-bar"
		>
			<!-- Score block -->
			<div class="flex shrink-0 items-center gap-4">
				<Score {score} size="md" />
			</div>

			<!-- Severity distribution -->
			<div class="flex min-w-0 flex-col gap-1.5">
				<div
					class="text-ink-faint flex items-baseline justify-between gap-2 font-mono text-[11px] tabular-nums"
				>
					<span class="tracking-wide uppercase">Severity distribution</span>
					<span class="text-ink-muted"
						>{(report.summary.totalIssues ?? 0).toLocaleString()} issue{report.summary
							.totalIssues === 1
							? ''
							: 's'}</span
					>
				</div>
				<SeverityBar counts={severityCounts} showLabels height="md" />
			</div>

			<!-- Page ratio + duration -->
			<div
				class="flex shrink-0 flex-wrap items-center gap-x-4 gap-y-1 font-mono text-xs tabular-nums sm:flex-col sm:items-end sm:gap-y-1"
			>
				<span class="text-ink-muted">
					<span class="text-ink font-semibold">{pagesWithIssues}</span>
					<span class="text-ink-faint">/</span>
					<span class="text-ink font-semibold">{pagesScanned}</span>
					<span class="text-ink-faint ml-1 tracking-wide uppercase">pages</span>
				</span>
				{#if duration}
					<span class="text-ink-muted">
						<span class="text-ink font-semibold">{duration}</span>
						<span class="text-ink-faint ml-1 tracking-wide uppercase">elapsed</span>
					</span>
				{/if}
			</div>
		</div>

		<!-- Triage strip + meta info -->
		{#if band || triageHeadline}
			<div
				class={cn(
					'border-line/70 bg-surface/70 flex items-center gap-3 rounded-lg border px-3 py-2'
				)}
				data-testid="report-triage-strip"
			>
				<AlertTriangle class="text-accent h-4 w-4 shrink-0" />
				<p class="text-ink-muted text-sm leading-snug">{triageHeadline}</p>
			</div>
		{/if}

		<!-- Secondary meta row (scan ID, timestamps) -->
		<div
			class="text-ink-faint flex flex-wrap items-center gap-x-4 gap-y-1 font-mono text-[11px] tabular-nums"
		>
			<span><span class="uppercase">id</span> {jobId}</span>
			{#if scannedAt}
				<span><span class="uppercase">scanned</span> {scannedAt}</span>
			{/if}
		</div>
	</div>
</Panel>
