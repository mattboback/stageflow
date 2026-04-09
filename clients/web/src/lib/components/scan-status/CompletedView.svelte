<script lang="ts">
	import type { ScanResult } from '$lib/types/scan';

	import { buildApiUrl } from '$lib/api/utils';
	import { buttonVariants } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { ArrowRight, CheckCircle2, ClipboardList, FileText } from 'lucide-svelte';

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

<div class="space-y-8">
	<div class="flex items-start gap-4 rounded-xl border border-emerald-100 bg-emerald-50/60 p-5">
		<div class="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-600 text-white">
			<CheckCircle2 class="h-6 w-6" />
		</div>
		<div class="min-w-0 flex-1">
			<h3 class="text-ink text-lg font-semibold">Scan complete</h3>
			<p class="text-ink-muted mt-1 text-sm">
				The unified report is ready. Start there to triage the most important findings, then use the sidebar for raw logs and artifacts if you need deeper diagnostics.
			</p>
			<div class="text-ink-muted mt-4 flex flex-wrap gap-3 text-sm">
				<span class="inline-flex items-center gap-2 rounded-full bg-white/70 px-3 py-1">
					<FileText class="text-accent h-4 w-4" />
					{totalPages}
					{pluralize(totalPages, 'page')} scanned
				</span>
				<span
					class="inline-flex items-center gap-2 rounded-full bg-white/70 px-3 py-1 tabular-nums"
				>
					<span class="h-2 w-2 rounded-full bg-amber-500"></span>
					{totalViolations}
					{pluralize(totalViolations, 'issue')} found
				</span>
			</div>
			{#if jobId}
				<div class="mt-5 flex flex-wrap gap-3">
					<a
						href={`/scan/${jobId}/report`}
						class={cn(buttonVariants({ variant: 'default' }), 'inline-flex gap-2')}
					>
						<ClipboardList class="h-4 w-4" />
						Open Unified Report
						<ArrowRight class="h-4 w-4" />
					</a>
					{#if jsonUrl}
						<a
							href={jsonUrl}
							target="_blank"
							rel="noopener noreferrer"
							class={cn(buttonVariants({ variant: 'outline' }), 'inline-flex gap-2')}
						>
							<FileText class="h-4 w-4" />
							Download JSON
						</a>
					{/if}
				</div>
			{/if}
		</div>
	</div>
</div>
