<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';

	import { Button, buttonVariants } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { ArrowLeft, RefreshCw } from 'lucide-svelte';

	interface Props {
		result: ScanResult | null;
	}

	let { result }: Props = $props();

	function handleRetry() {
		window.location.reload();
	}
</script>

<div class="border-severity-critical border-l-2 py-1 pl-5">
	<h3 class="font-display text-ink-strong text-2xl font-semibold tracking-[-0.01em]">
		Scan failed
	</h3>
	<p class="text-ink-muted mt-2 max-w-prose text-sm leading-relaxed">
		{result?.error || 'The worker process was terminated unexpectedly. Please try again.'}
	</p>
	{#if result?.last_stage}
		<p class="text-ink-faint mt-3 font-mono text-[11px] uppercase">
			stage: {result.last_stage}
		</p>
	{/if}
	{#if result?.error_details}
		<div class="border-line bg-surface-muted text-ink mt-4 rounded-md border p-3 text-xs">
			<div class="text-ink-faint mb-2 font-mono text-[10px] uppercase">Error details</div>
			<pre
				class="max-h-40 overflow-auto font-mono leading-relaxed whitespace-pre-wrap">{result.error_details}</pre>
		</div>
	{/if}
	<div class="mt-6 flex flex-wrap gap-3">
		<Button onclick={handleRetry} variant="outline" class="gap-2">
			<RefreshCw class="h-4 w-4" aria-hidden="true" />
			Retry
		</Button>
		<a href="/playground" class={cn(buttonVariants({ variant: 'default' }), 'gap-2')}>
			<ArrowLeft class="h-4 w-4" aria-hidden="true" />
			Run another scan
		</a>
	</div>
</div>
