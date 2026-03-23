<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';

	import { Button, buttonVariants } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { ArrowLeft, RefreshCw, XCircle } from 'lucide-svelte';

	interface Props {
		result: ScanResult | null;
	}

	let { result }: Props = $props();

	function handleRetry() {
		window.location.reload();
	}
</script>

<div class="flex flex-col items-center justify-center space-y-6 py-12 text-center">
	<div class="rounded-full bg-red-100 p-4 text-red-600">
		<XCircle class="h-10 w-10" />
	</div>
	<div>
		<h3 class="text-ink text-xl font-bold tracking-tight">Scan Failed</h3>
		<p class="text-ink-muted mt-2 max-w-md">
			{result?.error || 'The worker process was terminated unexpectedly. Please try again.'}
		</p>
		{#if result?.last_stage}
			<p class="text-ink-faint mt-2 text-xs font-semibold tracking-wide uppercase">
				Stage: {result.last_stage}
			</p>
		{/if}
		{#if result?.error_details}
			<div
				class="border-line bg-surface-muted text-ink mt-4 rounded-lg border p-3 text-left text-xs"
			>
				<div class="text-ink-faint mb-2 text-[10px] font-semibold tracking-wider uppercase">
					Error details
				</div>
				<pre
					class="max-h-40 overflow-auto font-mono leading-relaxed whitespace-pre-wrap">{result.error_details}</pre>
			</div>
		{/if}
	</div>
	<div class="flex gap-3">
		<Button onclick={handleRetry} variant="outline" class="gap-2">
			<RefreshCw class="h-4 w-4" />
			Retry
		</Button>
		<a href="/" class={cn(buttonVariants({ variant: 'default' }), 'gap-2')}>
			<ArrowLeft class="h-4 w-4" />
			Back Home
		</a>
	</div>
</div>
