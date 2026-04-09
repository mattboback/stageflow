<script lang="ts">
	import type { ScanResult, ScanStatus } from '$lib/types/scan';

	import { Button, Chip, Panel } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { ArrowRight, Clock, FileText, Loader2 } from 'lucide-svelte';

	import AutomationTab from './artifacts-sidebar/AutomationTab.svelte';
	import LogsTab from './artifacts-sidebar/LogsTab.svelte';
	import { buildExpirationLabel } from './artifacts-sidebar/expiration';

	interface Props {
		status: ScanStatus;
		result: ScanResult | null;
	}

	type ArtifactTab = 'logs' | 'automation';
	const TABS: ArtifactTab[] = ['logs', 'automation'];

	let { status, result }: Props = $props();

	let userOpen = $state(true);
	let activeTab = $state<ArtifactTab>('logs');

	const artifacts = $derived(resolveArtifacts(status, result));
	const isLockedOpen = $derived(status !== 'complete');
	const open = $derived(isLockedOpen ? true : userOpen);
	const expirationLabel = $derived(buildExpirationLabel(result?.updated_at ?? null));
	const reportUrl = $derived(result?.id ? `/scan/${result.id}/report` : null);

	function toggleOpen() {
		if (isLockedOpen) return;
		userOpen = !userOpen;
	}

	function resolveArtifacts(
		status: ScanStatus,
		result: ScanResult | null
	): ScanResult['artifacts'] | null {
		if (!result?.artifacts) return null;
		if (status === 'loading') return null;
		return result.artifacts;
	}
</script>

<Panel padding="none" rounded="xl" class="text-ink shadow-sm lg:sticky lg:top-28">
	<!-- Header -->
	<div class="bg-surface-muted flex flex-col space-y-1.5 p-6">
		<div class="flex items-center justify-between gap-3">
			<h3 class="flex items-center gap-2 text-lg leading-none font-semibold tracking-tight">
				<FileText class="text-accent h-5 w-5" /> Logs &amp; Artifacts
			</h3>
			<Button
				variant="ghost"
				size="sm"
				onclick={toggleOpen}
				disabled={isLockedOpen}
				class="text-xs font-semibold"
			>
				{isLockedOpen ? 'In Progress' : open ? 'Hide' : 'Show'}
			</Button>
		</div>
		<p class="text-ink-faint text-xs">
			{status === 'complete'
				? 'Use this rail for diagnostics after reviewing the report.'
				: 'Inspect live logs and automation assets while the scan runs.'}
		</p>
		{#if status === 'complete' && reportUrl}
			<a
				href={reportUrl}
				class="border-line bg-surface text-ink hover:border-accent hover:text-accent mt-3 inline-flex items-center justify-between rounded-xl border px-3 py-2 text-sm font-medium transition-colors"
			>
				<span>Review unified report first</span>
				<ArrowRight class="h-4 w-4" />
			</a>
		{/if}
		{#if artifacts}
			<div
				class="bg-surface/70 text-ink-muted mt-4 flex gap-2 rounded-full p-1 text-xs font-semibold"
			>
				{#each TABS as tab (tab)}
					<Chip
						as="button"
						type="button"
						interactive
						tone={activeTab === tab ? 'active' : 'ghost'}
						class={cn('flex-1 capitalize', activeTab !== tab && 'opacity-60 hover:opacity-100')}
						onclick={() => (activeTab = tab)}
					>
						{tab}
					</Chip>
				{/each}
			</div>
		{/if}
	</div>

	{#if open}
		<div class="space-y-4 p-6 pt-6">
			{#if !artifacts}
				<div class="text-ink-faint flex flex-col items-center justify-center py-8 text-center">
					<Loader2
						class={cn(
							'mb-2 h-8 w-8',
							status === 'failed' ? 'hidden' : 'text-ink-faint animate-spin'
						)}
					/>
					<p class="text-sm">
							{status === 'failed'
								? 'No artifacts generated'
								: status === 'complete'
									? 'Artifacts are still being attached'
									: 'Generating artifacts…'}
					</p>
				</div>
			{:else}
				{#if activeTab === 'logs'}
					<LogsTab {artifacts} />
				{:else}
					<AutomationTab jobId={result?.id ?? null} />
				{/if}

				<!-- Expiration Notice -->
				<div class="rounded-lg bg-blue-50 p-4 text-sm text-blue-800">
					<p class="flex items-start gap-2">
						<Clock class="mt-0.5 h-4 w-4 shrink-0" />
						<span>
							Artifacts expire automatically
							{expirationLabel ? ` on ${expirationLabel}.` : ' within 24 hours.'}
						</span>
					</p>
				</div>
			{/if}
		</div>
	{:else}
		<div class="text-ink-faint px-6 pb-6 text-sm">Expand to view logs and automation snippets.</div>
	{/if}
</Panel>
