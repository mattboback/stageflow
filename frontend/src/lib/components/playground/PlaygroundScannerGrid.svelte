<script lang="ts">
	import type { ScannerPreset } from '$lib/components/playground/scanner-presets';
	import type { ScannerSelection } from '$lib/types/scan';

	import { Chip, Label } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { Bot, Link2, Loader2, Search, Settings2, Shield, Zap } from 'lucide-svelte';

	interface Props {
		scanners: ScannerSelection[];
		isLoading: boolean;
		preset: ScannerPreset;
		onPresetChange: (preset: ScannerPreset) => void;
		onToggle: (scannerId: string) => void;
	}

	const { scanners, isLoading, preset, onPresetChange, onToggle }: Props = $props();

	const enabledScannerCount = $derived(scanners.filter((s) => s.enabled).length);

	// Scanner metadata for display
	const scannerMeta: Record<
		string,
		{ icon: typeof Shield; description: string; color: string; requiresConfig?: boolean }
	> = {
		axe: {
			icon: Shield,
			description: 'WCAG accessibility testing with axe-core engine',
			color: 'text-blue-600'
		},
		lighthouse: {
			icon: Zap,
			description: 'Performance, SEO, and best practices audits',
			color: 'text-amber-500'
		},
		'link-checker': {
			icon: Link2,
			description: 'Detect broken links and redirect chains',
			color: 'text-emerald-600'
		},
		'security-headers': {
			icon: Shield,
			description: 'HTTP security header analysis and scoring',
			color: 'text-violet-600'
		},
		seo: {
			icon: Search,
			description: 'Meta tags, headings, and SEO optimization',
			color: 'text-rose-500'
		},
		'ai-navigator': {
			icon: Bot,
			description: 'LLM-powered agent that navigates toward a goal',
			color: 'text-purple-600',
			requiresConfig: true
		}
	};

	function selectableSurfaceClass(base: string, isSelected: boolean) {
		return cn(
			base,
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
				>
					Coverage
				</Chip>
				<Chip
					as="button"
					type="button"
					tone={preset === 'quick' ? 'active' : 'muted'}
					interactive={true}
					aria-pressed={preset === 'quick'}
					onclick={() => onPresetChange('quick')}
				>
					Quick
				</Chip>
				<Chip
					as="button"
					type="button"
					tone={preset === 'custom' ? 'active' : 'muted'}
					interactive={true}
					aria-pressed={preset === 'custom'}
					onclick={() => onPresetChange('custom')}
				>
					Custom
				</Chip>
			</div>
			<p class="text-ink-muted mt-2 text-xs">
				Coverage runs multiple scanners and may take longer.
			</p>
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
					color: 'text-ink-muted'
				}}
				{@const Icon = meta.icon}
				<button
					type="button"
					aria-pressed={scanner.enabled}
					onclick={() => onToggle(scanner.id)}
					class={selectableSurfaceClass(
						'group relative flex items-start gap-3 rounded-2xl border p-4 text-left transition-all duration-200',
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
						<span class="text-ink-muted line-clamp-2 text-xs">
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
