<script lang="ts">
	import type { ContrastVerdict } from '$lib/stores/contrast-verdicts.svelte';

	import { Button, Input, Label, StatusPill } from '$lib/components/ui';
	import {
		WCAG_THRESHOLDS,
		type ContrastLevel,
		contrastRatioFromStrings,
		formatRatio,
		parseColor,
		requiredLevel
	} from '$lib/utils/contrast';
	import { formatTimestamp } from '$lib/utils/date';
	import { ArrowLeftRight } from 'lucide-svelte';

	interface Props {
		fg: string;
		bg: string;
		ruleId: string;
		largeText: boolean;
		verdict: ContrastVerdict | null;
		onFgChange: (value: string) => void;
		onBgChange: (value: string) => void;
		onSwap: () => void;
		onLargeTextChange: (value: boolean) => void;
		onRecord: (verdict: 'pass' | 'fail', ratio: number | null) => void;
		onClear: () => void;
	}

	let {
		fg,
		bg,
		ruleId,
		largeText,
		verdict,
		onFgChange,
		onBgChange,
		onSwap,
		onLargeTextChange,
		onRecord,
		onClear
	}: Props = $props();

	const ratio = $derived(contrastRatioFromStrings(fg, bg));
	const required = $derived(requiredLevel(ruleId));

	const levels: ContrastLevel[] = ['AA', 'AAA'];

	function swatchStyle(value: string): string {
		return parseColor(value) ? `background: ${value}` : '';
	}
</script>

<div class="space-y-5">
	<div class="flex flex-wrap items-end gap-3">
		<div class="min-w-0 flex-1 basis-40">
			<Label for="contrast-fg" class="text-xs font-semibold">Text color</Label>
			<div class="mt-1.5 flex items-center gap-2">
				<span
					class="border-line h-9 w-9 shrink-0 rounded-sm border"
					style={swatchStyle(fg)}
					aria-hidden="true"
				></span>
				<Input
					id="contrast-fg"
					value={fg}
					oninput={(e) => onFgChange(e.currentTarget.value)}
					placeholder="#1a1714"
					class="h-9 font-mono text-xs"
					autocomplete="off"
					spellcheck={false}
				/>
			</div>
		</div>
		<button
			type="button"
			onclick={onSwap}
			class="border-line text-ink-muted hover:text-ink bg-surface mb-1 rounded-sm border p-2 transition-colors"
			aria-label="Swap text and background colors"
			title="Swap colors"
		>
			<ArrowLeftRight class="h-3.5 w-3.5" />
		</button>
		<div class="min-w-0 flex-1 basis-40">
			<Label for="contrast-bg" class="text-xs font-semibold">Background color</Label>
			<div class="mt-1.5 flex items-center gap-2">
				<span
					class="border-line h-9 w-9 shrink-0 rounded-sm border"
					style={swatchStyle(bg)}
					aria-hidden="true"
				></span>
				<Input
					id="contrast-bg"
					value={bg}
					oninput={(e) => onBgChange(e.currentTarget.value)}
					placeholder="#faf9f7"
					class="h-9 font-mono text-xs"
					autocomplete="off"
					spellcheck={false}
				/>
			</div>
		</div>
	</div>

	<div class="flex flex-wrap items-center gap-x-6 gap-y-4">
		<div>
			<p class="section-tag">Measured ratio</p>
			<p class="stat-mono text-ink-strong mt-1 text-3xl font-semibold" aria-live="polite">
				{#if ratio !== null}
					{formatRatio(ratio)}<span class="text-ink-faint text-lg"> : 1</span>
				{:else}
					—
				{/if}
			</p>
		</div>
		<ul class="space-y-1.5">
			{#each levels as level (level)}
				{@const threshold = WCAG_THRESHOLDS[level][largeText ? 'large' : 'normal']}
				{@const passes = ratio !== null && ratio >= threshold}
				<li class="flex items-center gap-2.5">
					<span
						class={[
							'stat-mono w-28 text-xs',
							level === required ? 'text-ink font-bold' : 'text-ink-muted'
						]}
					>
						{level} · {threshold.toFixed(1)}:1
					</span>
					{#if ratio !== null}
						<StatusPill
							tone={passes ? 'strong' : 'failing'}
							label={passes ? 'Pass' : 'Fail'}
							size="sm"
						/>
					{:else}
						<StatusPill tone="neutral" label="No colors" size="sm" />
					{/if}
					{#if level === required}
						<span class="text-ink-faint font-mono text-[10px] tracking-wider uppercase"
							>required</span
						>
					{/if}
				</li>
			{/each}
		</ul>
		<label class="text-ink-muted flex items-center gap-2 text-xs">
			<input
				type="checkbox"
				checked={largeText}
				onchange={(e) => onLargeTextChange(e.currentTarget.checked)}
				class="accent-accent h-3.5 w-3.5"
			/>
			Large text (≥24px, or bold ≥18.7px)
		</label>
	</div>

	<div class="border-line border-t pt-4">
		{#if verdict}
			<div class="flex flex-wrap items-center gap-3">
				<StatusPill
					tone={verdict.verdict === 'pass' ? 'strong' : 'failing'}
					label={`Verified · ${verdict.verdict}`}
					size="md"
				/>
				<span class="stat-mono text-ink-muted text-xs">
					{verdict.fg || '—'} on {verdict.bg || '—'}{verdict.ratio !== null
						? ` · ${formatRatio(verdict.ratio)}:1`
						: ''} · {formatTimestamp(verdict.at) ?? verdict.at}
				</span>
				<Button variant="ghost" size="sm" onclick={onClear}>Clear verdict</Button>
			</div>
		{:else}
			<div class="flex flex-wrap items-center gap-3">
				<p class="text-ink-muted mr-1 text-sm">Your judgement:</p>
				<Button variant="default" size="sm" onclick={() => onRecord('pass', ratio)}>
					Mark as pass
				</Button>
				<Button variant="destructive" size="sm" onclick={() => onRecord('fail', ratio)}>
					Mark as fail
				</Button>
			</div>
			<p class="text-ink-faint mt-2 text-xs">
				Recorded in this browser only — verdicts don't change the stored report.
			</p>
		{/if}
	</div>
</div>
