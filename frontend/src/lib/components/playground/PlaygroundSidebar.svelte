<script lang="ts">
	import { Panel } from '$lib/components/ui';
	import { Camera, ChartBar, CheckCircle2, FileText, Shield, Wrench } from 'lucide-svelte';
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
		<div class="px-5 py-5">
			<div class="relative">
				<!-- Vertical connector line -->
				<div class="absolute top-3 bottom-3 left-3 w-px bg-accent/15"></div>

				<div class="space-y-4">
					{#each [
						{ step: '01', label: 'Configure input', desc: 'Enter URLs or upload a ZIP.' },
						{ step: '02', label: 'Select scanners', desc: 'Choose checks to run.' },
						{ step: '03', label: 'Run scan', desc: 'Watch scan progress in real time.' },
						{ step: '04', label: 'Review output', desc: 'Inspect findings and evidence.' }
					] as item (item.step)}
						<div class="relative flex gap-3">
							<span
								class="text-accent bg-accent/10 stat-mono relative z-10 inline-flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-semibold"
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
			</div>
		</div>
	</Panel>

	<Panel
		padding="none"
		rounded="2xl"
		class="text-ink border-line/80 overflow-hidden shadow-[var(--shadow-sm)]"
	>
		<div class="border-line/80 bg-surface-muted/70 border-b px-5 py-4">
			<h3 class="text-sm font-semibold tracking-wide uppercase">What You'll Get</h3>
		</div>
		<div class="space-y-3 px-5 py-5">
			{#each [
				{ icon: FileText, label: 'Unified issue report', bg: 'bg-blue-50', color: 'text-blue-600' },
				{ icon: Camera, label: 'Annotated screenshots', bg: 'bg-amber-50', color: 'text-amber-600' },
				{ icon: Wrench, label: 'Remediation guidance', bg: 'bg-emerald-50', color: 'text-emerald-600' },
				{ icon: ChartBar, label: 'Lighthouse scores', bg: 'bg-violet-50', color: 'text-violet-600' }
			] as item (item.label)}
				{@const ItemIcon = item.icon}
				<div class="flex items-center gap-3">
					<div class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg {item.bg}">
						<ItemIcon class="h-3.5 w-3.5 {item.color}" />
					</div>
					<span class="text-sm text-ink-muted">{item.label}</span>
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

	<div class="bg-accent-soft/50 border-accent/10 flex items-center gap-3 rounded-2xl border px-4 py-3 shadow-[var(--shadow-xs)]">
		<Shield class="text-accent h-5 w-5 shrink-0" />
		<div>
			<p class="text-sm font-medium">Secure &amp; Private</p>
			<p class="text-ink-muted text-xs">Your data is never stored or shared</p>
		</div>
	</div>
</div>
