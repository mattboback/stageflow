<script lang="ts">
	import type { ScanResult, ScanStatus } from '$lib/types/scan';

	import { Panel } from '$lib/components/ui';

	import CompletedView from './CompletedView.svelte';
	import FailedView from './FailedView.svelte';
	import ProcessingView from './ProcessingView.svelte';
	import StatusStepper from './StatusStepper.svelte';

	interface Props {
		status: ScanStatus;
		result: ScanResult | null;
		logs: string[];
	}

	const { status, result, logs }: Props = $props();

	const isComplete = $derived(status === 'complete');
	const isFailed = $derived(status === 'failed' || status === 'error');
</script>

<Panel padding="none" rounded="xl" class="text-ink overflow-hidden shadow-sm">
	<div class="border-line bg-surface-muted border-b px-6 pt-8 pb-10">
		<StatusStepper currentStatus={status} />
	</div>
	<div class="p-8">
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
</Panel>
