<script lang="ts">
import type { ReportError } from "$lib/types/unified-report";

import { Panel } from "$lib/components/ui";

interface Props {
	errors: ReportError[] | undefined;
}

let { errors }: Props = $props();

const list = $derived(errors ?? []);
</script>

<Panel class="shadow-sm" padding="none" rounded="2xl">
	<div class="border-line border-b p-4">
		<h3 class="text-ink text-base leading-none font-semibold tracking-tight">Report Errors</h3>
	</div>
	{#if list.length === 0}
		<div class="text-ink-muted p-6 text-center text-sm">No errors reported.</div>
	{:else}
		<div class="divide-line divide-y">
			{#each list as error, idx (idx)}
				<div class="p-4 text-sm">
					<p class="text-ink font-semibold">{error.message}</p>
					<p class="text-ink-muted mt-1">
						{error.code} · {error.scope}
						{#if error.scannerId}
							· Scanner: {error.scannerId}
						{/if}
						{#if error.pageId}
							· Page: {error.pageId}
						{/if}
					</p>
					<p class="text-ink-muted mt-2 text-xs">
						{error.retryable ? 'Retryable' : 'Not retryable'}
					</p>
				</div>
			{/each}
		</div>
	{/if}
</Panel>
