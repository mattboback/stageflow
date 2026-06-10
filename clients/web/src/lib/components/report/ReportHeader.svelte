<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';
	import type { UnifiedReport } from '$lib/types/unified-report';

	import { buildApiUrl } from '$lib/api/utils';
	import { Button, Panel, Score, SeverityBar } from '$lib/components/ui';
	import { groupIssuesByRule } from '$lib/report';
	import { cn, formatDuration, formatTimestamp } from '$lib/utils';
	import {
		AlertTriangle,
		ArrowRight,
		CheckCircle2,
		ExternalLink,
		FileSearch,
		Layers3,
		RefreshCw
	} from 'lucide-svelte';

	interface Props {
		jobId: string;
		report: UnifiedReport;
		job: ScanResult | null;
		onJumpToIssues?: (severity?: string) => void;
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
	const patternCount = $derived(groupIssuesByRule(report.issues ?? []).length);

	interface Triage {
		message: string;
		cta: { label: string; severity?: string } | null;
		containerClass: string;
		buttonClass: string;
		clean: boolean;
		prominent: boolean;
	}

	const triage = $derived.by((): Triage => {
		if (criticalCount > 0) {
			return {
				message: `Start with the ${criticalCount.toLocaleString()} critical issue${criticalCount === 1 ? '' : 's'} — these block users outright.`,
				cta: {
					label: `Fix these ${criticalCount.toLocaleString()} critical issue${criticalCount === 1 ? '' : 's'}`,
					severity: 'critical'
				},
				containerClass: 'border-red-200 bg-red-50 text-red-900',
				buttonClass: 'bg-red-600 hover:bg-red-700',
				clean: false,
				prominent: true
			};
		}
		if (seriousCount > 0) {
			return {
				message: `No critical findings. Prioritize the ${seriousCount.toLocaleString()} serious issue${seriousCount === 1 ? '' : 's'} next.`,
				cta: {
					label: `Fix these ${seriousCount.toLocaleString()} serious issue${seriousCount === 1 ? '' : 's'}`,
					severity: 'serious'
				},
				containerClass: 'border-orange-200 bg-orange-50 text-orange-900',
				buttonClass: 'bg-orange-500 hover:bg-orange-600',
				clean: false,
				prominent: true
			};
		}
		if (report.summary.totalIssues > 0) {
			return {
				message: `No critical or serious findings — work through the remaining ${patternCount.toLocaleString()} issue pattern${patternCount === 1 ? '' : 's'}.`,
				cta: { label: 'Review all issues' },
				containerClass: 'border-amber-200 bg-amber-50 text-amber-900',
				buttonClass: 'bg-amber-500 hover:bg-amber-600',
				clean: false,
				prominent: false
			};
		}
		if ((report.errors?.length ?? 0) > 0) {
			return {
				message: 'No issues detected, but scan errors need review before treating this as clean.',
				cta: null,
				containerClass: 'border-amber-200 bg-amber-50 text-amber-900',
				buttonClass: '',
				clean: false,
				prominent: false
			};
		}
		return {
			message: 'No issues detected. Spot-check pages and artifacts to confirm release readiness.',
			cta: null,
			containerClass: 'border-emerald-200 bg-emerald-50 text-emerald-900',
			buttonClass: '',
			clean: true,
			prominent: false
		};
	});
</script>

<Panel class="border-line/60 bg-surface mb-6 border shadow-xs" padding="lg" rounded="2xl">
	<div class="flex flex-col gap-4">
		<!-- Top row: title + actions -->
		<div class="flex flex-wrap items-start justify-between gap-3">
			<div class="min-w-0 flex-1">
				<p class="text-ink-faint text-[11px] font-semibold tracking-[0.12em] uppercase">
					Scan report
				</p>
				{#if report.meta.baseUrl}
					<h1
						class="text-ink mt-1 text-xl leading-tight font-bold tracking-tight break-all sm:text-2xl"
						data-testid="report-header-title"
					>
						<a
							href={report.meta.baseUrl}
							target="_blank"
							rel="noopener noreferrer"
							class="hover:text-accent group inline-flex items-baseline gap-1.5 transition-colors"
						>
							{report.meta.baseUrl}
							<ExternalLink
								class="text-ink-faint group-hover:text-accent h-4 w-4 shrink-0 self-center transition-colors"
								aria-hidden="true"
							/>
						</a>
					</h1>
				{:else}
					<h1
						class="text-ink mt-1 text-xl leading-tight font-bold tracking-tight sm:text-2xl"
						data-testid="report-header-title"
					>
						Scan <span class="font-mono">#{jobId.slice(0, 8)}</span>
					</h1>
				{/if}
			</div>
			<div class="flex flex-wrap items-center gap-2">
				{#if onJumpToIssues}
					<Button variant="default" size="sm" onclick={() => onJumpToIssues()} class="gap-1.5">
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
						title="Per-scanner standalone report"
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

		<!-- Triage strip: the report's primary next action -->
		<div
			class={cn(
				'flex flex-wrap items-center justify-between gap-x-4 gap-y-2 rounded-xl border px-4',
				triage.prominent ? 'py-4' : 'py-3',
				triage.containerClass
			)}
			data-testid="report-triage-strip"
		>
			<div class="flex min-w-0 items-center gap-3">
				{#if triage.clean}
					<CheckCircle2 class="h-5 w-5 shrink-0 text-emerald-600" aria-hidden="true" />
				{:else}
					<AlertTriangle class="h-5 w-5 shrink-0 opacity-80" aria-hidden="true" />
				{/if}
				<p class={cn('leading-snug font-medium', triage.prominent ? 'text-base' : 'text-sm')}>
					{triage.message}
				</p>
			</div>
			{#if triage.cta && onJumpToIssues}
				{@const cta = triage.cta}
				<button
					type="button"
					onclick={() => onJumpToIssues(cta.severity)}
					class={cn(
						'inline-flex shrink-0 items-center gap-1.5 rounded-lg font-semibold text-white shadow-xs transition-colors',
						triage.prominent ? 'px-5 py-2.5 text-sm' : 'px-3.5 py-2 text-xs',
						triage.buttonClass
					)}
					data-testid="triage-cta"
				>
					{cta.label}
					<ArrowRight class="h-3.5 w-3.5" aria-hidden="true" />
				</button>
			{/if}
		</div>

		<!-- Condensed stats bar -->
		<div
			class="grid grid-cols-1 gap-x-6 gap-y-4 sm:grid-cols-[auto_1fr_auto] sm:items-center"
			data-testid="report-stats-bar"
		>
			<!-- Score block -->
			<div class="flex shrink-0 items-center gap-4">
				<Score {score} grade={report.summary.scoreGrade ?? null} size="md" />
			</div>

			<!-- Severity distribution -->
			<div class="flex min-w-0 flex-col gap-1.5">
				<div
					class="text-ink-faint flex items-baseline justify-between gap-2 font-mono text-[11px] tabular-nums"
				>
					<span class="tracking-wide uppercase">Severity distribution</span>
					<span class="text-ink-muted"
						>{patternCount.toLocaleString()} pattern{patternCount === 1 ? '' : 's'} · {(
							report.summary.totalIssues ?? 0
						).toLocaleString()} occurrence{report.summary.totalIssues === 1 ? '' : 's'}</span
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
