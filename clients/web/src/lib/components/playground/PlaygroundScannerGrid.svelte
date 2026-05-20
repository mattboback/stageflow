<script lang="ts">
	import type { ScannerPreset } from '$lib/domain/scanners/presets';
	import type { ScannerSelection } from '$lib/types/scan';

	import { Chip, Label } from '$lib/components/ui';
	import { SCANNER_META } from '$lib/report';
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

	// Brand-specific color specs for premium glows and icons
	const BRAND_COLORS: Record<string, { rgb: string; color: string; bg: string }> = {
		axe: { rgb: '13,92,99', color: '#0d5c63', bg: '#e6f4f4' },
		lighthouse: { rgb: '245,158,11', color: '#f59e0b', bg: '#fffbeb' },
		'security-headers': { rgb: '139,92,246', color: '#8b5cf6', bg: '#f5f3ff' },
		seo: { rgb: '244,63,94', color: '#f43f5e', bg: '#fff1f2' },
		'link-checker': { rgb: '16,185,129', color: '#10b981', bg: '#f0fdf4' },
		'ai-navigator': { rgb: '217,70,239', color: '#d946ef', bg: '#fdf4ff' },
		'open-graph': { rgb: '6,182,212', color: '#06b6d4', bg: '#ecfeff' },
		'spelling-grammar': { rgb: '101,163,13', color: '#65a30d', bg: '#f7fee7' }
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
						{@const brand = BRAND_COLORS[scanner.id] || { color: '#0d5c63', bg: '#e6f4f4' }}
						{#if meta}
							{@const MiniIcon = meta.icon}
							<span
								class="bg-surface text-ink-muted inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium"
							>
								<span
									class="mr-0.5 inline-block h-1.5 w-1.5 rounded-full"
									style="background-color: {brand.color}"
								></span>
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
		<div class="scanner-grid-container">
			<div class="scanner-grid">
				{#each scanners as scanner (scanner.id)}
					{@const meta = SCANNER_META[scanner.id] || {
						icon: Shield,
						label: scanner.id.replace(/-/g, ' '),
						description: 'Run scan checks'
					}}
					{@const brand = BRAND_COLORS[scanner.id] || {
						rgb: '13,92,99',
						color: '#0d5c63',
						bg: '#e6f4f4'
					}}
					{@const Icon = meta.icon}
					<button
						type="button"
						aria-pressed={scanner.enabled}
						onclick={() => onToggle(scanner.id)}
						class={cn(
							'scanner-grid-card group relative flex items-start gap-3 rounded-2xl border p-4 text-left transition-[all] duration-300',
							'focus-visible:ring-accent focus-visible:ring-offset-paper focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none',
							scanner.enabled
								? 'bg-surface -translate-y-0.5 scale-[1.01] border-transparent'
								: 'border-line bg-surface hover:border-accent/30 hover:bg-surface-muted hover:-translate-y-0.5 hover:shadow-sm'
						)}
						style={scanner.enabled
							? `box-shadow: 0 10px 25px -5px rgba(${brand.rgb}, 0.12), 0 8px 10px -6px rgba(${brand.rgb}, 0.12), 0 0 0 1px rgba(${brand.rgb}, 0.35); border-color: rgba(${brand.rgb}, 0.4);`
							: ''}
					>
						<div
							class={cn(
								'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl transition-[background-color,color,transform,box-shadow] duration-300 group-hover:scale-105',
								scanner.enabled ? 'text-white' : 'bg-surface-muted text-ink-muted'
							)}
							style={scanner.enabled
								? `background-color: ${brand.color}; box-shadow: 0 4px 12px rgba(${brand.rgb}, 0.35);`
								: ''}
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
							style={scanner.enabled
								? `background-color: ${brand.color}; border-color: ${brand.color};`
								: ''}
						>
							{#if scanner.enabled}
								{@render checkMarkSvg()}
							{/if}
						</div>
					</button>
				{/each}
			</div>
		</div>
	{/if}
</div>
