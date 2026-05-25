<script lang="ts">
	import type { ScannerSelection } from '$lib/types/scan';

	import { Chip } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import { AlertTriangle, CheckCircle2, Lock } from 'lucide-svelte';

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

<aside class="space-y-4 xl:sticky xl:top-24" aria-label="Scan summary">
	<div class="border-line bg-surface rounded-2xl border shadow-[var(--shadow-sm)]">
		<header class="border-line/80 flex items-center justify-between gap-3 border-b px-5 py-4">
			<div>
				<h2 class="text-ink text-sm font-semibold tracking-[-0.005em]">Run summary</h2>
				<p class="text-ink-muted mt-0.5 text-xs">Everything queued for this scan.</p>
			</div>
			<Chip tone={canSubmit ? 'success' : 'warning'} size="sm">
				{canSubmit ? 'Ready' : 'Needs input'}
			</Chip>
		</header>

		<dl class="border-line/80 divide-line/80 divide-y border-b text-sm">
			<div class="flex items-center justify-between gap-3 px-5 py-3">
				<dt class="text-ink-muted text-xs font-medium tracking-[0.04em] uppercase">Target</dt>
				<dd class="text-ink font-medium">{mode === 'url' ? 'Live URLs' : 'ZIP archive'}</dd>
			</div>
			<div class="flex items-center justify-between gap-3 px-5 py-3">
				<dt class="text-ink-muted text-xs font-medium tracking-[0.04em] uppercase">Scanners</dt>
				<dd class="text-ink font-medium">
					{enabledScanners.length}<span class="text-ink-faint"> / {scanners.length}</span>
				</dd>
			</div>
			<div class="flex items-center justify-between gap-3 px-5 py-3">
				<dt class="text-ink-muted text-xs font-medium tracking-[0.04em] uppercase">Screenshots</dt>
				<dd class="text-ink font-medium">{screenshot ? 'Included' : 'Off'}</dd>
			</div>
			<div class="flex items-center justify-between gap-3 px-5 py-3">
				<dt class="text-ink-muted text-xs font-medium tracking-[0.04em] uppercase">Highlights</dt>
				<dd class="text-ink font-medium capitalize">{highlightStyle}</dd>
			</div>
			{#if isAiNavigatorEnabled}
				<div class="flex items-center justify-between gap-3 px-5 py-3">
					<dt class="text-ink-muted text-xs font-medium tracking-[0.04em] uppercase">
						AI Navigator
					</dt>
					<dd class={cn('font-medium', isAiConfigValid ? 'text-emerald-700' : 'text-amber-700')}>
						{isAiConfigValid ? 'Configured' : 'Needs objective'}
					</dd>
				</div>
			{/if}
			{#if isAuthEnabled}
				<div class="flex items-center justify-between gap-3 px-5 py-3">
					<dt class="text-ink-muted text-xs font-medium tracking-[0.04em] uppercase">
						Authentication
					</dt>
					<dd class={cn('font-medium', isAuthConfigValid ? 'text-emerald-700' : 'text-amber-700')}>
						{isAuthConfigValid ? 'Configured' : 'Incomplete'}
					</dd>
				</div>
			{/if}
		</dl>

		{#if previewScanners.length > 0}
			<div class="border-line/80 border-b px-5 py-4">
				<p class="text-ink-muted mb-2 text-xs font-medium tracking-[0.04em] uppercase">Selected</p>
				<div class="flex flex-wrap gap-1.5">
					{#each previewScanners as scanner (scanner.id)}
						<span
							class="border-line bg-surface-muted/60 text-ink inline-flex rounded-md border px-2 py-0.5 text-xs font-medium capitalize"
						>
							{scanner.id.replace(/-/g, ' ')}
						</span>
					{/each}
					{#if enabledScanners.length > previewScanners.length}
						<span
							class="border-line bg-surface-muted/60 text-ink-muted inline-flex rounded-md border px-2 py-0.5 text-xs font-medium"
						>
							+{enabledScanners.length - previewScanners.length} more
						</span>
					{/if}
				</div>
			</div>
		{/if}

		<div class="px-5 py-4">
			{#if canSubmit}
				<div class="flex items-start gap-2.5">
					<CheckCircle2 class="mt-0.5 h-4 w-4 shrink-0 text-emerald-600" aria-hidden="true" />
					<div class="min-w-0">
						<p class="text-ink text-sm font-semibold">Ready to start scan</p>
						<p class="text-ink-muted mt-1 text-xs leading-relaxed">
							Your scan launches immediately and streams progress on the next screen.
						</p>
					</div>
				</div>
			{:else}
				<div class="flex items-start gap-2.5">
					<AlertTriangle class="mt-0.5 h-4 w-4 shrink-0 text-amber-600" aria-hidden="true" />
					<div class="min-w-0">
						<p class="text-ink text-sm font-semibold">Complete these before running</p>
						<ul class="text-ink-muted mt-1.5 space-y-1 text-xs leading-relaxed">
							{#each missingRequirements as requirement (requirement)}
								<li>{requirement}</li>
							{/each}
						</ul>
					</div>
				</div>
			{/if}
		</div>
	</div>

	<div
		class="border-line bg-surface flex items-start gap-3 rounded-2xl border px-4 py-3 shadow-[var(--shadow-xs)]"
	>
		<Lock class="text-ink-faint mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
		<div>
			<p class="text-ink text-sm font-medium">What happens next</p>
			<p class="text-ink-muted mt-0.5 text-xs leading-relaxed">
				You go straight to live scan status, then into the unified report with screenshots and
				remediation details.
			</p>
		</div>
	</div>
</aside>
