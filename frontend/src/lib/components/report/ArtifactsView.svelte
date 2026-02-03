<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';

	import { buildApiUrl } from '$lib/api/utils';
	import { Button, Panel } from '$lib/components/ui';
	import { formatTimestamp } from '$lib/utils';
	import { ExternalLink, RefreshCw } from 'lucide-svelte';

	interface Props {
		jobId: string;
		job: ScanResult | null;
		onRefreshArtifacts?: () => void;
	}

	const { jobId, job, onRefreshArtifacts }: Props = $props();

	const aggregatedJsonUrl = $derived(jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/results`) : null);
	const aggregatedHtmlUrl = $derived(jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/report`) : null);
	const artifacts = $derived(job?.artifacts ?? null);
	const updatedLabel = $derived(formatTimestamp(job?.updated_at));
</script>

<div class="space-y-6">
	<Panel class="shadow-sm" padding="none" rounded="2xl">
		<div class="border-line border-b p-4">
			<div class="flex items-center justify-between gap-3">
				<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
					Aggregated Report Links
				</h3>
				{#if onRefreshArtifacts}
					<Button variant="outline" size="sm" onclick={onRefreshArtifacts} class="gap-2">
						<RefreshCw class="h-4 w-4" />
						Refresh links
					</Button>
				{/if}
			</div>
		</div>
		<div class="space-y-2 p-4 text-sm">
			{#if aggregatedJsonUrl}
				<a
					href={aggregatedJsonUrl}
					target="_blank"
					rel="noopener noreferrer"
					class="text-accent inline-flex items-center gap-2 hover:underline"
				>
					Aggregated JSON
					<ExternalLink class="h-3 w-3" />
				</a>
			{/if}
			{#if aggregatedHtmlUrl}
				<a
					href={aggregatedHtmlUrl}
					target="_blank"
					rel="noopener noreferrer"
					class="text-accent inline-flex items-center gap-2 hover:underline"
				>
					Primary HTML report
					<ExternalLink class="h-3 w-3" />
				</a>
			{/if}
			{#if artifacts?.scan_stage_log}
				<a
					href={artifacts.scan_stage_log}
					target="_blank"
					rel="noopener noreferrer"
					class="text-accent inline-flex items-center gap-2 hover:underline"
				>
					Scan stage log
					<ExternalLink class="h-3 w-3" />
				</a>
			{/if}
			{#if artifacts?.scan_recipe}
				<a
					href={artifacts.scan_recipe}
					target="_blank"
					rel="noopener noreferrer"
					class="text-accent inline-flex items-center gap-2 hover:underline"
				>
					Scan recipe
					<ExternalLink class="h-3 w-3" />
				</a>
			{/if}
			{#if artifacts?.extraction_stage_log}
				<a
					href={artifacts.extraction_stage_log}
					target="_blank"
					rel="noopener noreferrer"
					class="text-accent inline-flex items-center gap-2 hover:underline"
				>
					Extraction log
					<ExternalLink class="h-3 w-3" />
				</a>
			{/if}
			{#if artifacts?.extraction_recipe}
				<a
					href={artifacts.extraction_recipe}
					target="_blank"
					rel="noopener noreferrer"
					class="text-accent inline-flex items-center gap-2 hover:underline"
				>
					Extraction recipe
					<ExternalLink class="h-3 w-3" />
				</a>
			{/if}
			<div class="text-ink-muted text-xs">
				Links are signed and can expire. Refresh to regenerate if needed.
				{#if updatedLabel}
					<span> Last updated {updatedLabel}.</span>
				{/if}
			</div>
		</div>
	</Panel>

	<Panel class="shadow-sm" padding="none" rounded="2xl">
		<div class="border-line border-b p-4">
			<h3 class="text-ink text-base leading-none font-semibold tracking-tight">
				Scanner Artifacts
			</h3>
		</div>
		<div class="p-4">
			{#if !artifacts?.scanner_artifacts}
				<p class="text-ink-muted text-sm">No scanner artifacts available yet.</p>
			{:else}
				<div class="space-y-4">
					{#each Object.entries(artifacts.scanner_artifacts) as [scannerId, item] (scannerId)}
						<Panel padding="sm" rounded="xl">
							<p class="text-ink font-semibold">{scannerId}</p>
							<div class="mt-2 flex flex-wrap gap-3 text-sm">
								{#if item.results_json}
									<a
										href={item.results_json}
										target="_blank"
										rel="noopener noreferrer"
										class="text-accent inline-flex items-center gap-2 hover:underline"
									>
										JSON results
										<ExternalLink class="h-3 w-3" />
									</a>
								{/if}
								{#if item.report_html}
									<a
										href={item.report_html}
										target="_blank"
										rel="noopener noreferrer"
										class="text-accent inline-flex items-center gap-2 hover:underline"
									>
										HTML report
										<ExternalLink class="h-3 w-3" />
									</a>
								{/if}
								{#if item.scan_stage_log}
									<a
										href={item.scan_stage_log}
										target="_blank"
										rel="noopener noreferrer"
										class="text-accent inline-flex items-center gap-2 hover:underline"
									>
										Stage log
										<ExternalLink class="h-3 w-3" />
									</a>
								{/if}
								{#if item.scan_recipe}
									<a
										href={item.scan_recipe}
										target="_blank"
										rel="noopener noreferrer"
										class="text-accent inline-flex items-center gap-2 hover:underline"
									>
										Scan recipe
										<ExternalLink class="h-3 w-3" />
									</a>
								{/if}
							</div>
						</Panel>
					{/each}
				</div>
			{/if}
		</div>
	</Panel>
</div>
