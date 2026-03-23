<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { buildApiUrl } from '$lib/api/utils';
	import { Button, Panel } from '$lib/components/ui';
	import { cn, formatDuration, formatTimestamp } from '$lib/utils';
	import { ExternalLink, RefreshCw } from 'lucide-svelte';

	interface Props {
		jobId: string;
		report: UnifiedReport;
		job: ScanResult | null;
		onRefreshArtifacts?: () => void;
	}

	let { jobId, report, job: _job, onRefreshArtifacts }: Props = $props();

	const jsonUrl = $derived(jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/results`) : null);
	const htmlUrl = $derived(jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/report`) : null);

	const scannedAt = $derived(formatTimestamp(report.meta.scannedAt));
	const completedAt = $derived(formatTimestamp(report.meta.completedAt));
	const duration = $derived(formatDuration(report.meta.durationMs));
	const pagesScanned = $derived(report.summary.pagesScanned ?? 0);
	const pagesWithIssues = $derived(report.summary.pagesWithIssues ?? 0);
	const criticalCount = $derived(report.summary.bySeverity?.critical ?? 0);
	const affectedRatio = $derived.by(() => {
		if (pagesScanned <= 0) return 0;
		return Math.round((pagesWithIssues / pagesScanned) * 100);
	});
</script>

<Panel
	class="border-line/70 from-surface via-accent-mist/75 to-accent-subtle/65 mb-6 overflow-hidden border bg-gradient-to-br shadow-sm"
	padding="lg"
	rounded="3xl"
>
	<div
		class="bg-accent/18 pointer-events-none absolute -top-20 -right-20 h-56 w-56 rounded-full blur-3xl"
	></div>
	<div class="flex flex-row items-start justify-between gap-4">
		<div class="min-w-0 flex-1">
			<p class="text-ink-muted text-xs font-semibold tracking-[0.12em] uppercase">Scan report</p>
			{#if report.meta.baseUrl}
				<h1 class="text-ink mt-2 text-2xl leading-tight font-bold break-all sm:text-3xl">
					{report.meta.baseUrl}
				</h1>
			{/if}
			<div class="mt-3 flex flex-wrap items-center gap-2">
				<span
					class="border-line bg-surface text-ink-muted rounded-full border px-2.5 py-1 text-[11px] font-semibold tracking-wide uppercase"
				>
					{affectedRatio}% pages impacted
				</span>
				{#if criticalCount > 0}
					<span
						class="rounded-full border border-red-200 bg-red-50 px-2.5 py-1 text-[11px] font-semibold tracking-wide text-red-700 uppercase"
					>
						{criticalCount} critical
					</span>
				{/if}
			</div>
			<div class="mt-4 flex flex-wrap items-center gap-2 text-xs">
				{#if scannedAt}
					<span class="bg-surface-muted text-ink-muted rounded-md px-2.5 py-1"
						>Scanned {scannedAt}</span
					>
				{/if}
				{#if completedAt}
					<span class="bg-surface-muted text-ink-muted rounded-md px-2.5 py-1">
						Completed {completedAt}
					</span>
				{/if}
				{#if duration}
					<span class="bg-surface-muted text-ink-muted rounded-md px-2.5 py-1"
						>Duration {duration}</span
					>
				{/if}
			</div>
			<div class="mt-4 flex flex-wrap items-center gap-2.5">
				{#if jsonUrl}
					<a
						href={jsonUrl}
						target="_blank"
						rel="noopener noreferrer"
						class="border-line text-ink hover:border-accent hover:text-accent inline-flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors"
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
						class="border-line text-ink hover:border-accent hover:text-accent inline-flex items-center gap-2 rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors"
					>
						HTML report
						<ExternalLink class="h-3 w-3" />
					</a>
				{/if}
				{#if onRefreshArtifacts}
					<Button variant="outline" size="sm" onclick={onRefreshArtifacts} class="gap-2">
						<RefreshCw class="h-4 w-4" />
						Refresh
					</Button>
				{/if}
			</div>
		</div>
		{#if report.summary.score !== undefined}
			<div
				class={cn(
					'flex h-24 w-24 shrink-0 flex-col items-center justify-center rounded-full border text-center font-bold shadow-sm ring-4 ring-white/60',
					report.summary.score >= 90
						? 'border-emerald-200 bg-emerald-100 text-emerald-700'
						: report.summary.score >= 70
							? 'border-amber-200 bg-amber-100 text-amber-700'
							: 'border-red-200 bg-red-100 text-red-700'
				)}
			>
				<span class="text-[1.75rem] leading-none">
					{report.summary.scoreGrade ?? Math.round(report.summary.score)}
				</span>
				<span class="mt-1 text-[0.6rem] font-semibold tracking-[0.14em] uppercase">Score</span>
			</div>
		{/if}
	</div>
</Panel>
