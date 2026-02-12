<script lang="ts">
	import { Panel } from '$lib/components/ui';
	import { CheckCircle2, Shield } from 'lucide-svelte';
</script>

<div class="space-y-4 xl:sticky xl:top-24">
	<Panel
		padding="none"
		rounded="2xl"
		class="text-ink border-line/80 overflow-hidden shadow-[var(--shadow-sm)]"
	>
		<div class="border-line/80 bg-surface-muted/70 border-b px-5 py-4">
			<h3 class="text-sm font-semibold tracking-wide uppercase">How It Works</h3>
		</div>
		<div class="space-y-3.5 px-5 py-5">
			{#each [
				{ step: '01', label: 'Configure input', desc: 'Enter URLs or upload a ZIP.' },
				{ step: '02', label: 'Select scanners', desc: 'Choose checks to run.' },
				{ step: '03', label: 'Run scan', desc: 'Watch scan progress in real time.' },
				{ step: '04', label: 'Review output', desc: 'Inspect findings and evidence.' }
			] as item (item.step)}
				<div class="flex gap-3">
					<span
						class="text-accent bg-accent/10 stat-mono inline-flex h-6 w-8 shrink-0 items-center justify-center rounded-md text-xs font-semibold"
					>
						{item.step}
					</span>
					<div>
						<p class="text-sm font-medium">{item.label}</p>
						<p class="text-ink-muted text-xs">{item.desc}</p>
					</div>
				</div>
			{/each}
		</div>
	</Panel>

	<Panel
		padding="none"
		rounded="2xl"
		class="text-ink border-line/80 overflow-hidden shadow-[var(--shadow-sm)]"
	>
		<div class="border-line/80 bg-surface-muted/70 border-b px-5 py-4">
			<h3 class="text-sm font-semibold tracking-wide uppercase">Included</h3>
		</div>
		<div class="px-5 py-5">
			<ul class="space-y-2.5">
				{#each [
					'WCAG 2.1 AA/AAA testing',
					'Lighthouse performance audits',
					'Broken link detection',
					'Security header analysis',
					'Screenshot with highlights',
					'Remediation guidance',
					'AI navigation testing'
				] as feature (feature)}
					<li class="flex items-center gap-2 text-sm">
						<CheckCircle2 class="h-3.5 w-3.5 shrink-0 text-emerald-600" />
						<span class="text-ink-muted">{feature}</span>
					</li>
				{/each}
			</ul>
		</div>
	</Panel>

	<div class="border-line/80 bg-surface/80 flex items-center gap-3 rounded-2xl border px-4 py-3 shadow-[var(--shadow-xs)]">
		<Shield class="text-accent h-5 w-5 shrink-0" />
		<div>
			<p class="text-sm font-medium">Secure &amp; Private</p>
			<p class="text-ink-muted text-xs">Your data is never stored or shared</p>
		</div>
	</div>
</div>
