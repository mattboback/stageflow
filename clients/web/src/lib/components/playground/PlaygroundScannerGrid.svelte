<script lang="ts">
	import type { ScannerSelection } from '$lib/types/scan';

	import { Label } from '$lib/components/ui';
	import { SCANNER_META } from '$lib/report';
	import { cn } from '$lib/utils';
	import { Check, Loader2, Settings2, Shield } from 'lucide-svelte';

	interface Props {
		scanners: ScannerSelection[];
		isLoading: boolean;
		loadError?: string | null;
		onToggle: (scannerId: string) => void;
	}

	let { scanners, isLoading, loadError = null, onToggle }: Props = $props();

	const enabledScannerCount = $derived(scanners.filter((s) => s.enabled).length);
</script>

<div>
	<div class="mb-3 flex items-center justify-between">
		<Label class="text-sm font-semibold">Scanners</Label>
		{#if !isLoading && scanners.length > 0}
			<span class="text-ink-muted font-mono text-[11px] tabular-nums">
				<span class="text-accent font-semibold">{enabledScannerCount}</span> / {scanners.length} enabled
			</span>
		{/if}
	</div>

	{#if isLoading}
		<div
			class="text-ink-muted border-line flex items-center justify-center gap-2 rounded-md border py-8"
		>
			<Loader2 class="text-accent h-5 w-5 animate-spin" aria-hidden="true" />
			<span class="text-sm">Loading available scanners…</span>
		</div>
	{:else if loadError}
		<p class="text-ink-muted border-line rounded-md border px-4 py-8 text-center text-sm">
			{loadError}
		</p>
	{:else if scanners.length === 0}
		<p class="text-ink-muted border-line rounded-md border py-8 text-center text-sm">
			No scanners available
		</p>
	{:else}
		<ul class="border-line divide-line/70 divide-y rounded-md border">
			{#each scanners as scanner (scanner.id)}
				{@const meta = SCANNER_META[scanner.id] || {
					icon: Shield,
					label: scanner.id.replace(/-/g, ' '),
					description: 'Run scan checks'
				}}
				<li>
					<button
						type="button"
						aria-pressed={scanner.enabled}
						onclick={() => onToggle(scanner.id)}
						class={cn(
							'hover:bg-surface-muted flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors',
							'focus-visible:ring-accent focus-visible:ring-2 focus-visible:outline-none focus-visible:ring-inset'
						)}
					>
						<span
							class={cn(
								'flex h-4 w-4 shrink-0 items-center justify-center rounded-sm border transition-colors',
								scanner.enabled ? 'border-accent bg-accent text-white' : 'border-line bg-surface'
							)}
							aria-hidden="true"
						>
							{#if scanner.enabled}
								<Check class="h-3 w-3" strokeWidth={3} />
							{/if}
						</span>
						<span class="text-ink min-w-0 shrink-0 text-sm font-medium">{meta.label}</span>
						{#if scanner.id === 'ai-navigator' && scanner.enabled}
							<span class="inline-flex shrink-0 items-center gap-1 text-[11px] text-amber-700">
								<Settings2 class="h-3 w-3" aria-hidden="true" />
								configure below
							</span>
						{/if}
						<span class="text-ink-faint ml-auto hidden min-w-0 truncate pl-3 text-xs md:inline">
							{meta.description}
						</span>
					</button>
				</li>
			{/each}
		</ul>
	{/if}
</div>
