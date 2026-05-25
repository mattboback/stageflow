<script lang="ts">
	import type { ScannerSelection } from '$lib/types/scan';

	import { Chip, Panel } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { CheckCircle2, Lock, Play, Sparkles, Target, WandSparkles } from 'lucide-svelte';

	interface Props {
		mode: 'url' | 'zip';
		scanners: ScannerSelection[];
		screenshot: boolean;
		highlightStyle: 'solid' | 'dashed';
		canSubmit: boolean;
		isAiNavigatorEnabled: boolean;
		isAiConfigValid: boolean;
		isAuthEnabled: boolean;
		isAuthConfigValid: boolean;
		missingRequirements: string[];
	}

	let {
		mode,
		scanners,
		screenshot,
		highlightStyle,
		canSubmit,
		isAiNavigatorEnabled,
		isAiConfigValid,
		isAuthEnabled,
		isAuthConfigValid,
		missingRequirements
	}: Props = $props();

	const enabledScanners = $derived(scanners.filter((scanner) => scanner.enabled));
	const previewScanners = $derived(enabledScanners.slice(0, 4));
</script>

<div class="space-y-4 xl:sticky xl:top-24">
	<Panel
		padding="none"
		rounded="2xl"
		class="text-ink border-line/80 overflow-hidden shadow-[var(--shadow-sm)]"
	>
		<div class="border-line/80 bg-surface-muted/70 border-b px-5 py-4">
			<div class="flex items-center justify-between gap-3">
				<div>
					<h3 class="text-sm font-semibold tracking-wide uppercase">Run Summary</h3>
					<p class="text-ink-muted mt-1 text-xs">Everything needed to launch this scan.</p>
				</div>
				<Chip tone={canSubmit ? 'success' : 'warning'} size="sm">
					{canSubmit ? 'Ready' : 'Needs input'}
				</Chip>
			</div>
		</div>
		<div class="px-5 py-5">
			<div class="space-y-4">
				<div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
					<div class="bg-surface-muted/60 rounded-2xl px-4 py-3">
						<div
							class="text-ink-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.14em] uppercase"
						>
							<Target class="h-3.5 w-3.5" />
							Target
						</div>
						<p class="mt-2 text-sm font-medium">{mode === 'url' ? 'Live URLs' : 'ZIP archive'}</p>
					</div>
					<div class="bg-surface-muted/60 rounded-2xl px-4 py-3">
						<div
							class="text-ink-muted flex items-center gap-2 text-[11px] font-semibold tracking-[0.14em] uppercase"
						>
							<Sparkles class="h-3.5 w-3.5" />
							Scanners
						</div>
						<p class="mt-2 text-sm font-medium">
							{enabledScanners.length} enabled
							{#if enabledScanners.length > 0}
								<span class="text-ink-muted">of {scanners.length}</span>
							{/if}
						</p>
					</div>
				</div>

				{#if previewScanners.length > 0}
					<div>
						<p class="text-ink-muted mb-2 text-[11px] font-semibold tracking-[0.14em] uppercase">
							Selected now
						</p>
						<div class="flex flex-wrap gap-1.5">
							{#each previewScanners as scanner (scanner.id)}
								<span class="bg-surface inline-flex rounded-full px-2.5 py-1 text-xs font-medium">
									{scanner.id.replace(/-/g, ' ')}
								</span>
							{/each}
							{#if enabledScanners.length > previewScanners.length}
								<span
									class="text-ink-muted bg-surface inline-flex rounded-full px-2.5 py-1 text-xs font-medium"
								>
									+{enabledScanners.length - previewScanners.length} more
								</span>
							{/if}
						</div>
					</div>
				{/if}

				<div class="bg-surface-muted/60 rounded-2xl px-4 py-3">
					<p class="text-ink-muted text-[11px] font-semibold tracking-[0.14em] uppercase">
						Run options
					</p>
					<div class="mt-2 space-y-2 text-sm">
						<div class="flex items-center justify-between gap-3">
							<span class="text-ink-muted">Screenshots</span>
							<span class="font-medium">{screenshot ? 'Included' : 'Off'}</span>
						</div>
						<div class="flex items-center justify-between gap-3">
							<span class="text-ink-muted">Highlights</span>
							<span class="font-medium capitalize">{highlightStyle}</span>
						</div>
						{#if isAiNavigatorEnabled}
							<div class="flex items-center justify-between gap-3">
								<span class="text-ink-muted">AI Navigator</span>
								<span
									class={cn('font-medium', isAiConfigValid ? 'text-emerald-700' : 'text-amber-700')}
								>
									{isAiConfigValid ? 'Configured' : 'Needs objective'}
								</span>
							</div>
						{/if}
						{#if isAuthEnabled}
							<div class="flex items-center justify-between gap-3">
								<span class="text-ink-muted">Authentication</span>
								<span
									class={cn(
										'font-medium',
										isAuthConfigValid ? 'text-emerald-700' : 'text-amber-700'
									)}
								>
									{isAuthConfigValid ? 'Configured' : 'Incomplete'}
								</span>
							</div>
						{/if}
					</div>
				</div>

				<div
					class={cn(
						'rounded-2xl border px-4 py-3',
						canSubmit ? 'border-emerald-200 bg-emerald-50/70' : 'border-amber-200 bg-amber-50/70'
					)}
				>
					<div class="flex items-start gap-2.5">
						<div
							class={cn(
								'mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full',
								canSubmit ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'
							)}
						>
							{#if canSubmit}
								<Play class="h-4 w-4" />
							{:else}
								<WandSparkles class="h-4 w-4" />
							{/if}
						</div>
						<div class="min-w-0">
							<p class="text-sm font-semibold">
								{canSubmit ? 'Ready to start scan' : 'Complete these before running'}
							</p>
							{#if canSubmit}
								<p class="text-ink-muted mt-1 text-xs">
									Your scan will launch immediately and stream progress on the next screen.
								</p>
							{:else}
								<ul class="mt-1 space-y-1 text-xs">
									{#each missingRequirements as requirement (requirement)}
										<li class="flex items-start gap-2">
											<CheckCircle2 class="mt-0.5 h-3.5 w-3.5 shrink-0 text-amber-700" />
											<span>{requirement}</span>
										</li>
									{/each}
								</ul>
							{/if}
						</div>
					</div>
				</div>
			</div>
		</div>
	</Panel>

	<div
		class="bg-accent-soft/50 border-accent/10 flex items-center gap-3 rounded-2xl border px-4 py-3 shadow-[var(--shadow-xs)]"
	>
		<Lock class="text-accent h-5 w-5 shrink-0" />
		<div>
			<p class="text-sm font-medium">What happens next</p>
			<p class="text-ink-muted text-xs">
				You&apos;ll go straight to live scan status, then into the unified report with screenshots
				and remediation details.
			</p>
		</div>
	</div>
</div>
