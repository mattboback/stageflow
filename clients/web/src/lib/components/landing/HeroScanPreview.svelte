<script lang="ts">
	import TerminalCardHeader from '$lib/components/ui/TerminalCardHeader.svelte';

	const scannerRows = [
		{ name: 'accessibility', label: 'Axe', pct: 100, status: 'complete', color: 'bg-blue-500' },
		{ name: 'performance', label: 'Lighthouse', pct: 84, status: '84%', color: 'bg-amber-500' },
		{ name: 'crawlability', label: 'SEO', pct: 66, status: '66%', color: 'bg-rose-500' },
		{ name: 'security', label: 'Headers', pct: 42, status: '42%', color: 'bg-violet-500' }
	] as const;

	const findings = [
		{ severity: 'critical', count: 2, label: 'Critical', color: 'bg-red-500' },
		{ severity: 'serious', count: 8, label: 'Serious', color: 'bg-orange-400' },
		{ severity: 'moderate', count: 14, label: 'Moderate', color: 'bg-amber-400' }
	] as const;
</script>

<div class="hero-preview hidden lg:block" aria-hidden="true">
	<TerminalCardHeader path="stageflow scan --target https://stageflow.org" />

	<!-- Scanner progress -->
	<div
		class="space-y-3 border-b border-[var(--color-line)] px-5 pt-5 pb-3 font-mono text-xs leading-relaxed"
	>
		{#each scannerRows as scanner (scanner.name)}
			<div class="space-y-1.5">
				<div class="flex items-center justify-between gap-4">
					<span class="text-ink-muted text-[11px] font-medium">{scanner.label}</span>
					<span class="text-[11px]">
						{#if scanner.pct === 100}
							<span class="font-semibold text-emerald-600">✓ done</span>
						{:else}
							<span class="text-ink-faint">{scanner.status}</span>
						{/if}
					</span>
				</div>
				<div class="hero-preview-track">
					<div
						class={['hero-preview-fill transition-all duration-700', scanner.color].join(' ')}
						style="width: {scanner.pct}%"
					></div>
				</div>
			</div>
		{/each}
	</div>

	<!-- Mini results summary -->
	<div class="px-5 py-4">
		<div class="mb-3 flex items-center justify-between">
			<span class="text-ink-faint text-[10px] font-semibold tracking-wide uppercase">Results</span>
			<div class="flex items-center gap-1.5">
				<span
					class="inline-flex h-5 w-5 items-center justify-center rounded-full bg-amber-100 text-[10px] font-bold text-amber-700"
					>B</span
				>
				<span class="text-ink-faint text-[10px]">84/100</span>
			</div>
		</div>
		<div class="grid grid-cols-3 gap-2">
			{#each findings as f (f.severity)}
				<div class="rounded-lg bg-[var(--color-surface-muted)] px-2.5 py-2 text-center">
					<div class="mb-0.5 flex items-center justify-center gap-1">
						<span class={['h-1.5 w-1.5 shrink-0 rounded-full', f.color].join(' ')}></span>
						<span class="text-ink text-sm font-bold">{f.count}</span>
					</div>
					<span class="text-ink-faint text-[10px] font-medium">{f.label}</span>
				</div>
			{/each}
		</div>
		<p class="text-ink-faint mt-3 font-mono text-[10px] leading-snug">
			4 scanners · 3 pages · 24 issues found
		</p>
	</div>
</div>
