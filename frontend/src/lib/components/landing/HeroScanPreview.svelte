<script lang="ts">
	import TerminalCardHeader from '$lib/components/ui/TerminalCardHeader.svelte';
</script>

<div
	class="hidden rotate-1 rounded-2xl border border-line bg-surface shadow-lg lg:block"
	aria-hidden="true"
>
	<TerminalCardHeader path="stageflow scan" />
	<div class="space-y-3.5 px-5 py-4 font-mono text-xs leading-relaxed">
		{#each [
			{ name: 'axe', label: 'Accessibility', color: 'bg-blue-500', bg: 'bg-blue-100', pct: 100 },
			{ name: 'lighthouse', label: 'Lighthouse', color: 'bg-amber-500', bg: 'bg-amber-100', pct: 85 },
			{ name: 'seo', label: 'SEO', color: 'bg-rose-500', bg: 'bg-rose-100', pct: 65 },
			{ name: 'security', label: 'Security', color: 'bg-violet-500', bg: 'bg-violet-100', pct: 40 }
		] as scanner (scanner.name)}
			<div class="flex items-center gap-3">
				<span class="w-24 truncate text-ink-muted">{scanner.label}</span>
				<div class="h-1.5 flex-1 overflow-hidden rounded-full {scanner.bg}">
					<div
						class="h-full rounded-full {scanner.color}"
						style="width: {scanner.pct}%"
					></div>
				</div>
				<span class="text-ink-faint w-8 text-right">
					{#if scanner.pct === 100}
						<span class="text-emerald-600">done</span>
					{:else}
						{scanner.pct}%
					{/if}
				</span>
			</div>
		{/each}
		<div class="mt-1 border-t border-line/50 pt-2 text-ink-faint">
			<span class="text-accent">4</span> scanners running &middot; <span class="text-emerald-600">1</span> complete
		</div>
	</div>
</div>
