<script lang="ts">
	import { Panel } from '$lib/components/ui';
	import { CheckCircle2, Shield, Zap } from 'lucide-svelte';
</script>

<div class="space-y-6">
	<!-- How it works -->
	<Panel padding="none" rounded="xl" class="text-ink overflow-hidden shadow-sm">
		<div class="border-line from-surface-muted to-surface border-b bg-gradient-to-r p-6">
			<div class="flex items-center gap-3">
				<div class="bg-accent/10 flex h-8 w-8 items-center justify-center rounded-lg">
					<Zap class="text-accent h-4 w-4" />
				</div>
				<h3 class="text-base leading-none font-semibold tracking-tight">How It Works</h3>
			</div>
		</div>
		<div class="p-6 pt-4">
			<ol class="space-y-4">
				<li class="flex gap-3">
					<span
						class="bg-accent flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white shadow-sm"
						>1</span
					>
					<div>
						<span class="text-ink block text-sm font-medium">Configure input</span>
						<span class="text-ink-muted text-xs">Enter URLs or upload ZIP archive</span>
					</div>
				</li>
				<li class="flex gap-3">
					<span
						class="bg-accent flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white shadow-sm"
						>2</span
					>
					<div>
						<span class="text-ink block text-sm font-medium">Select scanners</span>
						<span class="text-ink-muted text-xs">Choose which checks to run</span>
					</div>
				</li>
				<li class="flex gap-3">
					<span
						class="bg-accent flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white shadow-sm"
						>3</span
					>
					<div>
						<span class="text-ink block text-sm font-medium">Run the scan</span>
						<span class="text-ink-muted text-xs">Watch real-time progress</span>
					</div>
				</li>
				<li class="flex gap-3">
					<span
						class="bg-accent flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-bold text-white shadow-sm"
						>4</span
					>
					<div>
						<span class="text-ink block text-sm font-medium">Review results</span>
						<span class="text-ink-muted text-xs">Get actionable insights</span>
					</div>
				</li>
			</ol>
		</div>
	</Panel>

	<!-- Features -->
	<Panel padding="none" rounded="xl" class="text-ink overflow-hidden shadow-sm">
		<div class="border-line from-surface-muted to-surface border-b bg-gradient-to-r p-6">
			<div class="flex items-center gap-3">
				<div class="bg-accent/10 flex h-8 w-8 items-center justify-center rounded-lg">
					<CheckCircle2 class="text-accent h-4 w-4" />
				</div>
				<h3 class="text-base leading-none font-semibold tracking-tight">Features</h3>
			</div>
		</div>
		<div class="p-6 pt-4">
			<ul class="space-y-3">
				<li class="flex items-center gap-2 text-sm">
					<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-500" />
					<span class="text-ink-muted">WCAG 2.1 AA/AAA testing</span>
				</li>
				<li class="flex items-center gap-2 text-sm">
					<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-500" />
					<span class="text-ink-muted">Lighthouse performance audits</span>
				</li>
				<li class="flex items-center gap-2 text-sm">
					<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-500" />
					<span class="text-ink-muted">Broken link detection</span>
				</li>
				<li class="flex items-center gap-2 text-sm">
					<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-500" />
					<span class="text-ink-muted">Security header analysis</span>
				</li>
				<li class="flex items-center gap-2 text-sm">
					<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-500" />
					<span class="text-ink-muted">Screenshot with issue highlights</span>
				</li>
				<li class="flex items-center gap-2 text-sm">
					<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-500" />
					<span class="text-ink-muted">Detailed remediation tips</span>
				</li>
				<li class="flex items-center gap-2 text-sm">
					<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-500" />
					<span class="text-ink-muted">AI-powered navigation testing</span>
				</li>
			</ul>
		</div>
	</Panel>

	<!-- Trust indicators -->
	<div class="border-line bg-surface-muted/50 rounded-xl border p-4">
		<div class="flex items-center gap-3">
			<div class="bg-surface flex h-10 w-10 items-center justify-center rounded-lg">
				<Shield class="text-accent h-5 w-5" />
			</div>
			<div>
				<p class="text-ink text-sm font-medium">Secure & Private</p>
				<p class="text-ink-muted text-xs">Your data is never stored or shared</p>
			</div>
		</div>
	</div>
</div>
