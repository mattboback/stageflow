<script lang="ts">
	import type { ScannerPreset } from '$lib/domain/scanners/presets';
	import type { ScannerSelection } from '$lib/types/scan';

	import { Chip, Label } from '$lib/components/ui';
	import { getScannerIconClass, getScannerTileClass, SCANNER_META } from '$lib/report';
	import { cn } from '$lib/utils';
	import { Loader2, Settings2, Shield, ShieldCheck, Zap } from 'lucide-svelte';

	interface Props {
		scanners: ScannerSelection[];
		isLoading: boolean;
		preset: ScannerPreset;
		onPresetChange: (preset: ScannerPreset) => void;
		onToggle: (scannerId: string) => void;
	}

	let { scanners, isLoading, preset, onPresetChange, onToggle }: Props = $props();

	const enabledScannerCount = $derived(scanners.filter((s) => s.enabled).length);
	const enabledScanners = $derived(scanners.filter((s) => s.enabled));

	const presetDescriptions: Record<ScannerPreset, string> = {
		coverage:
			'Recommended. Enables the full standard scanner set for the clearest release readout.',
		quick: 'Fastest pass. Runs axe only when you need a quick accessibility signal.',
		custom: 'Pick exactly which scanners to run for this target.'
	};
</script>

{#snippet checkMarkSvg()}
	<svg class="h-3 w-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
	</svg>
{/snippet}

<div>
	<div class="mb-3 flex items-center justify-between">
		<Label class="text-sm font-semibold">Select Scanners</Label>
		{#if !isLoading && scanners.length > 0}
			<span class="text-ink-muted text-xs font-medium">
				<span class="stat-mono text-accent">{enabledScannerCount}</span> / {scanners.length}
				enabled
			</span>
		{/if}
	</div>
	{#if !isLoading && scanners.length > 1}
		<div class="border-line/80 bg-surface-muted/55 mb-4 rounded-2xl border p-3">
			<p class="text-ink mb-2 text-xs font-semibold tracking-[0.12em] uppercase">Preset</p>
			<div class="flex flex-wrap gap-2">
				<Chip
					as="button"
					type="button"
					tone={preset === 'coverage' ? 'active' : 'muted'}
					interactive={true}
					aria-pressed={preset === 'coverage'}
					onclick={() => onPresetChange('coverage')}
					class="gap-1.5"
				>
					<ShieldCheck class="h-3.5 w-3.5" />
					Coverage
				</Chip>
				<Chip
					as="button"
					type="button"
					tone={preset === 'quick' ? 'active' : 'muted'}
					interactive={true}
					aria-pressed={preset === 'quick'}
					onclick={() => onPresetChange('quick')}
					class="gap-1.5"
				>
					<Zap class="h-3.5 w-3.5" />
					Quick
				</Chip>
				<Chip
					as="button"
					type="button"
					tone={preset === 'custom' ? 'active' : 'muted'}
					interactive={true}
					aria-pressed={preset === 'custom'}
					onclick={() => onPresetChange('custom')}
					class="gap-1.5"
				>
					<Settings2 class="h-3.5 w-3.5" />
					Custom
				</Chip>
			</div>
			<p class="text-ink-muted mt-2 text-xs">
				{presetDescriptions[preset]}
			</p>
			{#if preset !== 'custom' && enabledScanners.length > 0}
				<div class="mt-2 flex flex-wrap gap-1.5">
					{#each enabledScanners as scanner (scanner.id)}
						{@const meta = SCANNER_META[scanner.id]}
						{#if meta}
							{@const MiniIcon = meta.icon}
							<span
								class="bg-surface text-ink-muted inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium"
							>
								<MiniIcon class="h-2.5 w-2.5" />
								{meta.label}
							</span>
						{/if}
					{/each}
				</div>
			{/if}
		</div>
	{/if}
	{#if isLoading}
		<div
			class="text-ink-muted bg-surface-muted flex items-center justify-center gap-2 rounded-xl py-8"
		>
			<Loader2 class="text-accent h-5 w-5 animate-spin" />
			<span class="text-sm">Loading available scanners…</span>
		</div>
	{:else if scanners.length === 0}
		<p class="text-ink-muted bg-surface-muted rounded-xl py-8 text-center text-sm">
			No scanners available
		</p>
	{:else}
		<div class="grid gap-3 sm:grid-cols-2">
			{#each scanners as scanner (scanner.id)}
				{@const meta = SCANNER_META[scanner.id] || {
					icon: Shield,
					label: scanner.id.replace(/-/g, ' '),
					description: 'Run scan checks'
				}}
				{@const Icon = meta.icon}
				<button
					type="button"
					aria-pressed={scanner.enabled}
					onclick={() => onToggle(scanner.id)}
					class={cn(
						'group relative flex items-start gap-3 rounded-2xl border p-4 text-left transition-[background-color,border-color,box-shadow,transform] duration-200',
						'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-paper',
						getScannerTileClass(scanner.enabled)
					)}
				>
					<div
						class={cn(
							'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl transition-[background-color,color,transform] duration-200',
							getScannerIconClass(scanner.enabled)
						)}
					>
						<Icon class="h-5 w-5" />
					</div>
					<div class="min-w-0 flex-1">
						<span class="text-ink block text-sm font-semibold capitalize">
							{meta.label}
						</span>
						<span class="text-ink-muted text-xs">
							{meta.description}
						</span>
						{#if scanner.id === 'ai-navigator' && scanner.enabled}
							<span class="mt-1 inline-flex items-center gap-1 text-xs text-amber-600">
								<Settings2 class="h-3 w-3" />
								Configure below
							</span>
						{/if}
					</div>
					<div
						class={cn(
							'absolute top-3 right-3 flex h-5 w-5 items-center justify-center rounded-full border transition-[background-color,border-color,color] duration-200',
							scanner.enabled ? 'border-accent bg-accent text-white' : 'border-line bg-surface'
						)}
					>
						{#if scanner.enabled}
							{@render checkMarkSvg()}
						{/if}
					</div>
				</button>
			{/each}
		</div>
	{/if}
</div>
