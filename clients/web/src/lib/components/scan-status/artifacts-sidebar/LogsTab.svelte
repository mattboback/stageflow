<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';

	import OpenLinkButton from './OpenLinkButton.svelte';

	type Artifacts = NonNullable<ScanResult['artifacts']>;

	interface Props {
		artifacts: Artifacts;
	}

	let { artifacts }: Props = $props();

	const hasLogs = $derived(
		Boolean(
			artifacts.scan_stage_log ||
			artifacts.scan_recipe ||
			artifacts.extraction_stage_log ||
			artifacts.extraction_recipe
		)
	);
</script>

{#if hasLogs}
	<div class="space-y-2">
		<OpenLinkButton label="Scan Stage Log" url={artifacts.scan_stage_log} />
		<OpenLinkButton label="Scan Recipe" url={artifacts.scan_recipe} />
		<OpenLinkButton label="Extraction Stage Log" url={artifacts.extraction_stage_log} />
		<OpenLinkButton label="Extraction Recipe" url={artifacts.extraction_recipe} />
	</div>
{:else}
	<p class="text-ink-faint text-sm">No logs were generated for this run.</p>
{/if}
