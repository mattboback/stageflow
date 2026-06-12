<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';

	import { buildApiUrl } from '$lib/api/utils';
	import { buttonVariants } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { ArrowRight, ClipboardList, FileText } from 'lucide-svelte';

	interface Props {
		result: ScanResult | null;
	}

	let { result }: Props = $props();

	function pluralize(count: number, singular: string, plural = `${singular}s`) {
		return count === 1 ? singular : plural;
	}

	const totalViolations = $derived(result?.violations ?? 0);
	const totalPages = $derived(result?.progress?.total_pages ?? 0);
	const jobId = $derived(result?.id ?? '');
	const jsonUrl = $derived(jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/results`) : null);
</script>

<div class="border-accent border-l-2 py-1 pl-5">
	<h3 class="font-display text-ink-strong text-2xl font-semibold tracking-[-0.01em]">
		Scan complete
	</h3>
	<p class="text-ink-muted mt-2 max-w-prose text-sm leading-relaxed">
		The unified report is ready. Start there to triage the most important findings, then use the
		artifacts rail for raw logs and deeper diagnostics.
	</p>
	<p class="stat-mono text-ink mt-4 text-sm">
		{totalPages}
		{pluralize(totalPages, 'page')} scanned · {totalViolations}
		{pluralize(totalViolations, 'issue')} found
	</p>
	{#if jobId}
		<div class="mt-6 flex flex-wrap gap-3">
			<a
				href={`/scan/${jobId}/report`}
				class={cn(buttonVariants({ variant: 'default' }), 'inline-flex gap-2')}
			>
				<ClipboardList class="h-4 w-4" aria-hidden="true" />
				Open Unified Report
				<ArrowRight class="h-4 w-4" aria-hidden="true" />
			</a>
			{#if jsonUrl}
				<a
					href={jsonUrl}
					target="_blank"
					rel="noopener noreferrer"
					class={cn(buttonVariants({ variant: 'outline' }), 'inline-flex gap-2')}
				>
					<FileText class="h-4 w-4" aria-hidden="true" />
					Download JSON
				</a>
			{/if}
		</div>
	{/if}
</div>
