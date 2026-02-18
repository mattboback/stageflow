<script lang="ts">
	import TerminalCardHeader from '$lib/components/ui/TerminalCardHeader.svelte';

	const scannerRows = [
		{
			name: 'accessibility',
			label: 'Axe',
			pct: 100,
			status: 'complete',
			color: 'bg-blue-500'
		},
		{
			name: 'performance',
			label: 'Lighthouse',
			pct: 84,
			status: '84%',
			color: 'bg-amber-500'
		},
		{
			name: 'crawlability',
			label: 'SEO',
			pct: 66,
			status: '66%',
			color: 'bg-rose-500'
		},
		{
			name: 'security',
			label: 'Headers',
			pct: 42,
			status: '42%',
			color: 'bg-violet-500'
		}
	] as const;
</script>

<div class="hero-preview hidden lg:block" aria-hidden="true">
	<TerminalCardHeader path="stageflow scan --target https://stageflow.org" />
	<div class="space-y-4 px-5 py-5 font-mono text-xs leading-relaxed">
		{#each scannerRows as scanner (scanner.name)}
			<div class="space-y-1.5">
				<div class="flex items-center justify-between gap-4">
					<span class="text-ink-muted text-[11px] font-medium tracking-[0.08em] uppercase">
						{scanner.label}
					</span>
					<span class="text-ink-faint text-[11px]">
						{#if scanner.pct === 100}
							<span class="text-emerald-600">complete</span>
						{:else}
							{scanner.status}
						{/if}
					</span>
				</div>
				<div class="hero-preview-track">
					<div
						class={['hero-preview-fill', scanner.color].join(' ')}
						style="width: {scanner.pct}%"
					></div>
				</div>
			</div>
		{/each}

		<div class="border-line/60 grid grid-cols-2 gap-3 border-t pt-3">
			<div>
				<p class="hero-preview-legend">Events</p>
				<p class="text-ink mt-1 text-sm font-semibold">248 streamed</p>
			</div>
			<div>
				<p class="hero-preview-legend">Containers</p>
				<p class="text-ink mt-1 text-sm font-semibold">4 active</p>
			</div>
		</div>
	</div>
</div>
