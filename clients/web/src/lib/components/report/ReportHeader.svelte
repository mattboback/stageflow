<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { buildApiUrl } from '$lib/api/utils';
	import { Button, Panel } from '$lib/components/ui';
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
	const completedAt = $derived(formatTimestamp(report.meta.completedAt));
	const duration = $derived(formatDuration(report.meta.durationMs));
	const pagesScanned = $derived(report.summary.pagesScanned ?? 0);
	const pagesWithIssues = $derived(report.summary.pagesWithIssues ?? 0);
	const criticalCount = $derived(report.summary.bySeverity?.critical ?? 0);
	const score = $derived(report.summary.score ?? null);
	const scoreGrade = $derived(
		report.summary.scoreGrade ?? (score !== null ? String(Math.round(score)) : null)
	);
	const scoreBand = $derived.by(() => {
		if (score === null) return null;
		if (score >= 90) return { label: 'Strong', detail: 'A-range: release confidence is high.' };
		if (score >= 80)
			return { label: 'Watch', detail: 'B-range: review notable issues before shipping.' };
		if (score >= 70)
			return { label: 'Needs work', detail: 'C-range: multiple findings still affect quality.' };
		if (score >= 60)
			return { label: 'High risk', detail: 'D-range: remediation should precede release.' };
		return { label: 'Failing', detail: 'F-range: major quality risk remains.' };
	});
	const affectedRatio = $derived.by(() => {
		if (pagesScanned <= 0) return 0;
		return Math.round((pagesWithIssues / pagesScanned) * 100);
	});
	const seriousCount = $derived(report.summary.bySeverity?.serious ?? 0);
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
			<h1 class="text-ink mt-2 text-2xl leading-tight font-bold break-all sm:text-3xl">
				{report.meta.baseUrl ?? `Scan ${jobId}`}
			</h1>
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
				{#if onJumpToIssues}
					<Button variant="default" size="sm" onclick={onJumpToIssues} class="gap-2">
						<FileSearch class="h-4 w-4" />
						Review issues
					</Button>
				{/if}
				{#if onJumpToPages}
					<Button variant="outline" size="sm" onclick={onJumpToPages} class="gap-2">
						<Layers3 class="h-4 w-4" />
						Check top pages
					</Button>
				{/if}
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
			<div class="border-line/70 bg-surface/70 mt-4 rounded-2xl border px-4 py-3">
				<div class="flex items-start gap-3">
					<div
						class="bg-surface-muted text-accent flex h-9 w-9 shrink-0 items-center justify-center rounded-xl"
					>
						<AlertTriangle class="h-4 w-4" />
					</div>
					<div>
						<p class="text-ink text-sm font-semibold">Triage first</p>
						<p class="text-ink-muted mt-1 text-sm">{triageHeadline}</p>
					</div>
				</div>
			</div>
		</div>
		{#if score !== null}
			<div class="flex shrink-0 flex-col items-end gap-3">
				<div
					class={cn(
						'flex h-28 w-28 flex-col items-center justify-center rounded-full border-2 text-center font-bold shadow-md',
						score >= 90
							? 'border-emerald-200 bg-emerald-50 text-emerald-700'
							: score >= 70
								? 'border-amber-200 bg-amber-50 text-amber-700'
								: 'border-red-200 bg-red-50 text-red-700'
					)}
				>
					<span class="text-[2.25rem] leading-none font-bold">{scoreGrade}</span>
					<span class="mt-1.5 text-[0.6rem] font-bold tracking-[0.18em] uppercase opacity-70"
						>Score</span
					>
				</div>
				{#if scoreBand}
					<div
						class="bg-surface/85 border-line max-w-[14rem] rounded-2xl border px-3 py-2.5 text-right shadow-sm"
					>
						<p class="text-ink text-xs font-bold tracking-[0.1em] uppercase">
							{scoreBand.label}
						</p>
						<p class="text-ink-muted mt-1 text-xs leading-relaxed">{scoreBand.detail}</p>
						<p class="text-ink-faint mt-2 text-[10px]">A: 90+, B: 80-89, C: 70-79, D: 60-69, F: &lt;60</p>
					</div>
				{/if}
			</div>
		{/if}
	</div>
</Panel>
