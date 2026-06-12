<script lang="ts">
	import type { ScanResult, ScanStatus } from '$lib/types/scan';

	import CompletedView from './CompletedView.svelte';
	import FailedView from './FailedView.svelte';
	import ProcessingView from './ProcessingView.svelte';
	import StatusStepper from './StatusStepper.svelte';

	interface Props {
		status: ScanStatus;
		result: ScanResult | null;
		logs: string[];
	}

	let { status, result, logs }: Props = $props();

	const isComplete = $derived(status === 'complete');
	const isFailed = $derived(status === 'failed' || status === 'error');
</script>

<div class="border-line bg-surface text-ink rounded-md border">
	<div class="border-line border-b px-4 py-4 sm:px-6">
		<StatusStepper currentStatus={status} />
	</div>
	<div class="p-4 sm:p-6">
		{#if !isComplete && !isFailed}
			<ProcessingView {result} {logs} />
		{/if}

		{#if isComplete}
			<CompletedView {result} />
		{/if}

		{#if isFailed}
			<FailedView {result} />
		{/if}
	</div>
</div>
