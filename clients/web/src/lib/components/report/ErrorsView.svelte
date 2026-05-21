<script lang="ts">
	import type { ReportError } from '$lib/types/unified-report';

	import { Panel } from '$lib/components/ui';
	import { CheckCircle2 } from 'lucide-svelte';

	interface Props {
		errors: ReportError[] | undefined;
		/**
		 * `compact` collapses the zero-error state to a single success row
		 * instead of rendering a full-height empty card. Used inside the
		 * Diagnostics tab where a giant empty panel wastes half the layout.
		 */
		compact?: boolean;
	}

	let { errors, compact = false }: Props = $props();

	const list = $derived(errors ?? []);
</script>

{#if list.length === 0 && compact}
	<div
		class="border-line/70 bg-surface flex items-center gap-2.5 rounded-xl border px-3 py-2 text-sm shadow-xs"
		data-testid="errors-zero-compact"
	>
		<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-500" aria-hidden="true" />
		<span class="text-ink font-semibold">No report errors</span>
		<span class="text-ink-muted text-xs">All scanners completed without pipeline failures.</span>
	</div>
{:else}
	<Panel class="shadow-sm" padding="none" rounded="2xl">
		<div class="border-line border-b p-4">
			<h3 class="text-ink text-base leading-none font-semibold tracking-tight">Report Errors</h3>
		</div>
		{#if list.length === 0}
			<div class="text-ink-muted flex items-center gap-2 p-4 text-sm">
				<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-500" aria-hidden="true" />
				No errors reported.
			</div>
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
{/if}
