<script lang="ts">
	import type { ScannerPreset } from '$lib/domain/scanners/presets';

	import PlaygroundUrlInput from '$lib/components/playground/PlaygroundUrlInput.svelte';
	import PlaygroundZipUpload from '$lib/components/playground/PlaygroundZipUpload.svelte';
	import { Button, Chip } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import {
		AlertTriangle,
		ChevronDown,
		Loader2,
		Play,
		Settings2,
		ShieldCheck,
		Zap
	} from 'lucide-svelte';

	type Mode = 'url' | 'zip';

	interface Props {
		mode: Mode;
		urls: string;
		file: File | null;
		preset: ScannerPreset;
		hasPresets: boolean;
		enabledScannerCount: number;
		canSubmit: boolean;
		isSubmitting: boolean;
		missingRequirements: string[];
		advancedOpen: boolean;
		onModeChange: (mode: Mode) => void;
		onUrlsChange: (urls: string) => void;
		onNormalize: () => void;
		onFileChange: (file: File | null) => void;
		onFileError: (message: string) => void;
		onPresetChange: (preset: ScannerPreset) => void;
		onToggleAdvanced: () => void;
	}

	let {
		mode,
		urls,
		file,
		preset,
		hasPresets,
		enabledScannerCount,
		canSubmit,
		isSubmitting,
		missingRequirements,
		advancedOpen,
		onModeChange,
		onUrlsChange,
		onNormalize,
		onFileChange,
		onFileError,
		onPresetChange,
		onToggleAdvanced
	}: Props = $props();

	const presets = [
		{ id: 'coverage', label: 'Coverage', icon: ShieldCheck },
		{ id: 'quick', label: 'Quick', icon: Zap },
		{ id: 'custom', label: 'Custom', icon: Settings2 }
	] as const;

	function modeTabClass(selected: boolean): string {
		return cn(
			'-mb-px border-b-2 px-4 py-3 text-sm font-medium transition-colors',
			'focus-visible:ring-accent focus-visible:ring-2 focus-visible:outline-none focus-visible:ring-inset',
			selected
				? 'border-accent text-ink-strong'
				: 'text-ink-muted hover:text-ink border-transparent'
		);
	}
</script>

<div class="border-line bg-surface rounded-md border">
	<!-- Target mode tabs -->
	<div
		class="border-line flex items-center border-b px-2 sm:px-4"
		role="group"
		aria-label="Scan target type"
	>
		<button
			type="button"
			aria-pressed={mode === 'url'}
			class={modeTabClass(mode === 'url')}
			onclick={() => onModeChange('url')}
		>
			Live URLs
		</button>
		<button
			type="button"
			aria-pressed={mode === 'zip'}
			class={modeTabClass(mode === 'zip')}
			onclick={() => onModeChange('zip')}
		>
			ZIP Archive
		</button>
	</div>

	<!-- Target input -->
	<div class="p-4 sm:p-6">
		{#if mode === 'url'}
			<PlaygroundUrlInput {urls} {onUrlsChange} {onNormalize} />
		{:else}
			<PlaygroundZipUpload {file} {onFileChange} onError={onFileError} />
		{/if}
	</div>

	<!-- Action row: presets · count · submit -->
	<div class="border-line flex flex-wrap items-center gap-x-3 gap-y-3 border-t px-4 py-3.5 sm:px-6">
		{#if hasPresets}
			<div class="flex flex-wrap items-center gap-2" role="group" aria-label="Scanner preset">
				{#each presets as p (p.id)}
					{@const PresetIcon = p.icon}
					<Chip
						as="button"
						type="button"
						tone={preset === p.id ? 'active' : 'muted'}
						interactive={true}
						aria-pressed={preset === p.id}
						onclick={() => onPresetChange(p.id)}
						class="gap-1.5"
					>
						<PresetIcon class="h-3.5 w-3.5" aria-hidden="true" />
						{p.label}
					</Chip>
				{/each}
			</div>
		{/if}
		<span class="text-ink-faint ml-auto font-mono text-[11px] tabular-nums">
			{enabledScannerCount} scanner{enabledScannerCount === 1 ? '' : 's'}
		</span>
		<Button
			type="submit"
			variant="default"
			size="lg"
			class="h-10 gap-2 px-6 text-sm font-semibold"
			disabled={!canSubmit}
		>
			{#if isSubmitting}
				<Loader2 class="h-4 w-4 animate-spin" aria-hidden="true" />
				Starting…
			{:else}
				<Play class="h-4 w-4" aria-hidden="true" />
				Start Scan
			{/if}
		</Button>
	</div>

	<!-- What's blocking submission -->
	{#if !canSubmit && !isSubmitting && missingRequirements.length > 0}
		<div class="border-line border-t px-4 py-3 sm:px-6">
			<ul class="space-y-1">
				{#each missingRequirements as requirement (requirement)}
					<li class="text-ink-muted flex items-start gap-2 text-xs leading-relaxed">
						<AlertTriangle class="mt-0.5 h-3.5 w-3.5 shrink-0 text-amber-600" aria-hidden="true" />
						<span>{requirement}</span>
					</li>
				{/each}
			</ul>
		</div>
	{/if}

	<!-- Advanced options disclosure -->
	<button
		type="button"
		class="border-line text-ink-muted hover:text-ink flex w-full items-center gap-2 rounded-b-md border-t px-4 py-3 text-left text-sm font-medium transition-colors sm:px-6"
		aria-expanded={advancedOpen}
		aria-controls="advanced-options"
		onclick={onToggleAdvanced}
	>
		<ChevronDown
			class={cn('h-4 w-4 transition-transform', advancedOpen && 'rotate-180')}
			aria-hidden="true"
		/>
		Advanced options
		<span class="text-ink-faint ml-auto hidden font-mono text-[11px] sm:inline">
			scanners · screenshots · auth · AI
		</span>
	</button>
</div>
