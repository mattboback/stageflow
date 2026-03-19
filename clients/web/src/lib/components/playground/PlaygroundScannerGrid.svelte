<script lang="ts">
	import type { ScannerPreset } from '$lib/components/playground/scanner-presets';
	import type { ScannerSelection } from '$lib/types/scan';

	import { Chip, Label } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { Bot, Link2, Loader2, Search, Settings2, Shield, ShieldCheck, Zap } from 'lucide-svelte';

	interface Props {
		scanners: ScannerSelection[];
		isLoading: boolean;
		preset: ScannerPreset;
		onPresetChange: (preset: ScannerPreset) => void;
		onToggle: (scannerId: string) => void;
	}

	const { scanners, isLoading, preset, onPresetChange, onToggle }: Props = $props();

	const enabledScannerCount = $derived(scanners.filter((s) => s.enabled).length);
	const enabledScanners = $derived(scanners.filter((s) => s.enabled));

	// Scanner metadata for display
	const scannerMeta: Record<
		string,
		{ icon: typeof Shield; description: string; color: string; borderColor: string; requiresConfig?: boolean }
	> = {
		axe: {
			icon: Shield,
			description: 'WCAG accessibility testing with axe-core engine',
			color: 'text-blue-600',
			borderColor: 'border-l-blue-500'
		},
		lighthouse: {
			icon: Zap,
			description: 'Performance, SEO, and best practices audits',
			color: 'text-amber-500',
			borderColor: 'border-l-amber-500'
		},
		'link-checker': {
			icon: Link2,
			description: 'Detect broken links and redirect chains',
			color: 'text-emerald-600',
			borderColor: 'border-l-emerald-600'
		},
		'security-headers': {
			icon: Shield,
			description: 'HTTP security header analysis and scoring',
			color: 'text-violet-600',
			borderColor: 'border-l-violet-600'
		},
		seo: {
			icon: Search,
			description: 'Meta tags, headings, and SEO optimization',
			color: 'text-rose-500',
			borderColor: 'border-l-rose-500'
		},
		'ai-navigator': {
			icon: Bot,
			description: 'LLM-powered agent that navigates toward a goal',
			color: 'text-purple-600',
			borderColor: 'border-l-purple-600',
			requiresConfig: true
		}
	};

	const presetDescriptions: Record<ScannerPreset, string> = {
		coverage: 'All scanners except AI Navigator. Most thorough analysis.',
		quick: 'Axe accessibility scan only. Fastest results.',
		custom: 'Select individual scanners below.'
	};

	function selectableSurfaceClass(base: string, isSelected: boolean) {
		return cn(
			base,
			'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-paper',
			isSelected
				? 'border-accent/60 bg-accent/5 shadow-[0_0_0_1px_rgba(220,38,38,0.18)]'
				: 'border-line bg-surface hover:border-accent/30 hover:bg-surface-muted'
		);
	}
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
						{@const meta = scannerMeta[scanner.id]}
						{#if meta}
							{@const MiniIcon = meta.icon}
							<span class={cn('inline-flex items-center gap-1 rounded-full bg-surface px-2 py-0.5 text-[10px] font-medium', meta.color)}>
								<MiniIcon class="h-2.5 w-2.5" />
								{scanner.id.replace(/-/g, ' ')}
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
			<span class="text-sm">Loading available scanners...</span>
		</div>
	{:else if scanners.length === 0}
		<p class="text-ink-muted bg-surface-muted rounded-xl py-8 text-center text-sm">
			No scanners available
		</p>
	{:else}
		<div class="grid gap-3 sm:grid-cols-2">
			{#each scanners as scanner (scanner.id)}
				{@const meta = scannerMeta[scanner.id] || {
					icon: Shield,
					description: 'Run scan checks',
					color: 'text-ink-muted',
					borderColor: ''
				}}
				{@const Icon = meta.icon}
				<button
					type="button"
					aria-pressed={scanner.enabled}
					onclick={() => onToggle(scanner.id)}
					class={selectableSurfaceClass(
						cn(
							'group relative flex items-start gap-3 rounded-2xl border p-4 text-left transition-all duration-200',
							scanner.enabled && meta.borderColor ? `border-l-[3px] ${meta.borderColor}` : ''
						),
						scanner.enabled
					)}
				>
					<div
						class={cn(
							'flex h-10 w-10 shrink-0 items-center justify-center rounded-xl transition-all',
							scanner.enabled
								? 'bg-accent text-white'
								: 'bg-surface-muted text-ink-muted group-hover:bg-accent/10 group-hover:text-accent'
						)}
					>
						<Icon class="h-5 w-5" />
					</div>
					<div class="min-w-0 flex-1">
						<span class="text-ink block text-sm font-semibold capitalize">
							{scanner.id.replace(/-/g, ' ')}
						</span>
						<span class="text-ink-muted text-xs">
							{meta.description}
						</span>
						{#if meta.requiresConfig && scanner.enabled}
							<span class="mt-1 inline-flex items-center gap-1 text-xs text-amber-600">
								<Settings2 class="h-3 w-3" />
								Configure below
							</span>
						{/if}
					</div>
					<div
						class={cn(
							'absolute top-3 right-3 flex h-5 w-5 items-center justify-center rounded-full border transition-all',
							scanner.enabled
								? 'border-accent bg-accent text-white'
								: 'border-line bg-surface'
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
