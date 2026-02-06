<script lang="ts">
	import { Panel } from '$lib/components/ui';
	import { CheckCircle2, Shield } from 'lucide-svelte';
</script>

<div class="space-y-5">
	<!-- How it works -->
	<Panel padding="none" rounded="xl" class="text-ink overflow-hidden">
		<div class="border-line border-b p-5">
			<h3 class="text-sm font-semibold">How It Works</h3>
		</div>
		<div class="p-5">
			<ol class="space-y-4">
				{#each [
					{ step: '1', label: 'Configure input', desc: 'Enter URLs or upload a ZIP' },
					{ step: '2', label: 'Select scanners', desc: 'Choose which checks to run' },
					{ step: '3', label: 'Run the scan', desc: 'Watch real-time progress' },
					{ step: '4', label: 'Review results', desc: 'Get actionable insights' }
				] as item (item.step)}
					<li class="flex gap-3">
						<span
							class="bg-ink text-surface flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-xs font-bold"
						>{item.step}</span>
						<div>
							<span class="text-ink block text-sm font-medium">{item.label}</span>
							<span class="text-ink-muted text-xs">{item.desc}</span>
						</div>
					</li>
				{/each}
			</ol>
		</div>
	</Panel>

	<!-- Features -->
	<Panel padding="none" rounded="xl" class="text-ink overflow-hidden">
		<div class="border-line border-b p-5">
			<h3 class="text-sm font-semibold">Included</h3>
		</div>
		<div class="p-5">
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
						<CheckCircle2 class="h-3.5 w-3.5 shrink-0 text-emerald-500" />
						<span class="text-ink-muted">{feature}</span>
					</li>
				{/each}
			</ul>
		</div>
	</Panel>

	<!-- Trust -->
	<div class="flex items-center gap-3 rounded-xl border border-line p-4">
		<Shield class="text-ink-faint h-5 w-5 shrink-0" />
		<div>
			<p class="text-ink text-sm font-medium">Secure & Private</p>
			<p class="text-ink-muted text-xs">Your data is never stored or shared</p>
		</div>
	</div>
</div>
