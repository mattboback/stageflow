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

	const { jobId, report, job, onRefreshArtifacts }: Props = $props();

	const jsonUrl = $derived(jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/results`) : null);
	const htmlUrl = $derived(jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/report`) : null);

	const scannedAt = $derived(formatTimestamp(report.meta.scannedAt));
	const completedAt = $derived(formatTimestamp(report.meta.completedAt));
	const duration = $derived(formatDuration(report.meta.durationMs));
</script>

<Panel class="mb-8 shadow-sm" padding="lg" rounded="3xl">
	<div class="flex flex-wrap items-start justify-between gap-6">
		<div>
			<p class="text-ink-muted text-sm font-semibold tracking-wide uppercase">Scan report</p>
			{#if report.meta.baseUrl}
				<h1 class="text-ink mt-2 text-2xl font-bold">{report.meta.baseUrl}</h1>
			{/if}
			<div class="text-ink-muted mt-3 flex flex-wrap gap-4 text-xs">
				{#if scannedAt}
					<span>Scanned {scannedAt}</span>
				{/if}
				{#if completedAt}
					<span>Completed {completedAt}</span>
				{/if}
				{#if duration}
					<span>Duration {duration}</span>
				{/if}
				{#if job?.state}
					<span>State {job.state}</span>
				{/if}
			</div>
			<div class="mt-4 flex flex-wrap items-center gap-3">
				{#if jsonUrl}
					<a
						href={jsonUrl}
						target="_blank"
						rel="noopener noreferrer"
						class="text-accent inline-flex items-center gap-2 text-sm font-medium hover:underline"
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
						class="text-accent inline-flex items-center gap-2 text-sm font-medium hover:underline"
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
					'flex h-20 w-20 flex-col items-center justify-center rounded-full text-center text-xl font-bold',
					report.summary.score >= 90
						? 'bg-emerald-100 text-emerald-700'
						: report.summary.score >= 70
							? 'bg-amber-100 text-amber-700'
							: 'bg-red-100 text-red-700'
				)}
			>
				<span class="text-2xl">{report.summary.scoreGrade ?? Math.round(report.summary.score)}</span
				>
				<span class="text-xs font-semibold tracking-wide uppercase">Score</span>
			</div>
		{/if}
	</div>
</Panel>
