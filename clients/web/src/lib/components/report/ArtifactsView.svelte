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

	const aggregatedLinks = $derived.by(() => {
		const links: { href: string; label: string }[] = [];
		if (aggregatedJsonUrl) links.push({ href: aggregatedJsonUrl, label: 'Aggregated JSON' });
		if (aggregatedHtmlUrl) links.push({ href: aggregatedHtmlUrl, label: 'Primary HTML report' });
		if (artifacts?.scan_stage_log)
			links.push({ href: artifacts.scan_stage_log, label: 'Scan stage log' });
		if (artifacts?.scan_recipe) links.push({ href: artifacts.scan_recipe, label: 'Scan recipe' });
		if (artifacts?.extraction_stage_log)
			links.push({ href: artifacts.extraction_stage_log, label: 'Extraction log' });
		if (artifacts?.extraction_recipe)
			links.push({ href: artifacts.extraction_recipe, label: 'Extraction recipe' });
		return links;
	});
</script>

{#snippet artifactLink(href: string, label: string)}
	<a
		{href}
		target="_blank"
		rel="noopener noreferrer"
		class="text-accent inline-flex items-center gap-2 hover:underline"
	>
		{label}
		<ExternalLink class="h-3 w-3" />
	</a>
{/snippet}

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
			{#each aggregatedLinks as link (link.label)}
				{@render artifactLink(link.href, link.label)}
			{/each}
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
						{@const scannerLinks = [
							item.results_json && { href: item.results_json, label: 'JSON results' },
							item.report_html && { href: item.report_html, label: 'HTML report' },
							item.scan_stage_log && { href: item.scan_stage_log, label: 'Stage log' },
							item.scan_recipe && { href: item.scan_recipe, label: 'Scan recipe' }
						].filter((v): v is { href: string; label: string } => Boolean(v))}
						<Panel padding="sm" rounded="xl">
							<p class="text-ink font-semibold">{scannerId}</p>
							<div class="mt-2 flex flex-wrap gap-3 text-sm">
								{#each scannerLinks as link (link.label)}
									{@render artifactLink(link.href, link.label)}
								{/each}
							</div>
						</Panel>
					{/each}
				</div>
			{/if}
		</div>
	</Panel>
</div>
