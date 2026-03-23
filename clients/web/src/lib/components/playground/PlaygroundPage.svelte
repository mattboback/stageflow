<script lang="ts">
	import type { ScannerSelection } from '$lib/types/scan';

	import { goto } from '$app/navigation';
	import { fetchScanners, getDefaultScannerSelections, submitScanJob } from '$lib/api/client';
	import PlaygroundAiConfig from '$lib/components/playground/PlaygroundAiConfig.svelte';
	import PlaygroundHeroSection from '$lib/components/playground/PlaygroundHeroSection.svelte';
	import PlaygroundModeToggle from '$lib/components/playground/PlaygroundModeToggle.svelte';
	import PlaygroundOptions from '$lib/components/playground/PlaygroundOptions.svelte';
	import PlaygroundScannerGrid from '$lib/components/playground/PlaygroundScannerGrid.svelte';
	import PlaygroundSidebar from '$lib/components/playground/PlaygroundSidebar.svelte';
	import PlaygroundUrlInput from '$lib/components/playground/PlaygroundUrlInput.svelte';
	import PlaygroundZipUpload from '$lib/components/playground/PlaygroundZipUpload.svelte';
	import {
		buildAiNavigatorConfig,
		normalizeUrlListText,
		parseUrlList,
		validateHttpUrls
	} from '$lib/components/playground/playground-utils';
	import {
		type ScannerPreset,
		applyScannerPreset,
		detectScannerPreset
	} from '$lib/components/playground/scanner-presets';
	import { Alert, Button, Chip, PageSection, Panel } from '$lib/components/ui';
	import { AlertTriangle, CheckCircle2, Loader2, Play, Sparkles } from 'lucide-svelte';

	// Form state
	let mode = $state<'url' | 'zip'>('url');
	let urls = $state('');
	let file = $state<File | null>(null);
	let scanners = $state<ScannerSelection[]>([]);
	let scannerPreset = $state<ScannerPreset>('coverage');
	let screenshot = $state(true);
	let highlightStyle = $state<'solid' | 'dashed'>('solid');

	// UI state
	let isSubmitting = $state(false);
	let isLoadingScanners = $state(true);
	let error = $state<string | null>(null);
	let invalidUrls = $state<Array<{ url: string; reason: string }>>([]);

	// AI Navigator configuration
	let aiObjective = $state('');
	let aiModel = $state(import.meta.env.VITE_AI_NAVIGATOR_DEFAULT_MODEL || 'openai/gpt-4o-mini');
	let aiMaxSteps = $state(10);
	let aiMaxWallTimeMs = $state(120000);
	let aiInputValues = $state<Array<{ key: string; value: string }>>([]);
	let aiSuccessCriteria = $state<Array<{ type: string; value: string }>>([]);

	// Derived state
	const hasValidInput = $derived(mode === 'url' ? urls.trim().length > 0 : file !== null);
	const hasEnabledScanner = $derived(scanners.some((s) => s.enabled));
	const isAiNavigatorEnabled = $derived(scanners.some((s) => s.id === 'ai-navigator' && s.enabled));
	const isAiConfigValid = $derived(
		!isAiNavigatorEnabled || (aiObjective.trim().length > 0 && aiModel.trim().length > 0)
	);
	const canSubmit = $derived(
		hasValidInput && hasEnabledScanner && isAiConfigValid && !isSubmitting
	);

	// Load scanners on mount
	$effect(() => {
		fetchScanners()
			.then((data) => {
				scanners = getDefaultScannerSelections(data.scanners);
				scannerPreset = detectScannerPreset(scanners);
			})
			.catch((e) => {
				error = e instanceof Error ? e.message : 'Failed to load scanners';
			})
			.finally(() => {
				isLoadingScanners = false;
			});
	});

	// Update scanner config when AI settings change
	$effect(() => {
		if (isAiNavigatorEnabled && aiObjective.trim()) {
			const config = buildAiNavigatorConfig({
				objective: aiObjective,
				maxSteps: aiMaxSteps,
				maxWallTimeMs: aiMaxWallTimeMs,
				model: aiModel,
				inputValues: aiInputValues,
				successCriteria: aiSuccessCriteria
			});
			scanners = scanners.map((s) => (s.id === 'ai-navigator' ? { ...s, config } : s));
		}
	});

	function normalizeUrlsIfNeeded() {
		const { text, changed } = normalizeUrlListText(urls);
		if (!changed) return;
		urls = text;
	}

	function handleUrlsChange(newUrls: string) {
		urls = newUrls;
		invalidUrls = [];
	}

	function handleFileChange(newFile: File | null) {
		file = newFile;
		error = null;
	}

	function handleFileError(errorMsg: string) {
		error = errorMsg;
		file = null;
	}

	function handleScannerToggle(scannerId: string) {
		const nextScanners = scanners.map((s) =>
			s.id === scannerId ? { ...s, enabled: !s.enabled } : s
		);
		scanners = nextScanners;
		scannerPreset = 'custom';
	}

	function handlePresetChange(nextPreset: ScannerPreset) {
		scannerPreset = nextPreset;
		scanners = applyScannerPreset(scanners, nextPreset);
	}

	async function handleSubmit() {
		if (!canSubmit) return;

		isSubmitting = true;
		error = null;
		invalidUrls = [];

		try {
			if (mode === 'url') {
				normalizeUrlsIfNeeded();
			}

			const urlList = mode === 'url' ? parseUrlList(urls) : [];
			const { valid, invalid } = validateHttpUrls(urlList);
			if (mode === 'url' && invalid.length > 0) {
				error = 'Some URLs look invalid. Fix them and try again.';
				invalidUrls = invalid;
				return;
			}

			const result = await submitScanJob({
				mode,
				urls: mode === 'url' ? valid : urlList,
				file,
				scanners,
				screenshot,
				highlightStyle
			});

			await goto(`/scan/${result.job_id}`);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to submit scan';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<PlaygroundHeroSection />

<PageSection class="playground-shell relative overflow-hidden py-10 lg:py-14">
	<div
		class="via-accent/8 pointer-events-none absolute -top-24 left-1/2 h-72 w-[92vw] max-w-5xl -translate-x-1/2 rounded-[3rem] bg-gradient-to-r from-white/90 to-orange-200/20 blur-3xl"
	></div>

	<div class="container-width relative">
		<div class="grid items-start gap-7 xl:grid-cols-[minmax(0,1fr)_21rem]">
			<div>
				<Panel
					padding="none"
					rounded="2xl"
					class="text-ink border-line/80 overflow-hidden shadow-[var(--shadow-md)]"
				>
					<div class="border-line/80 bg-surface-muted/70 border-b px-6 py-5">
						<div class="flex flex-wrap items-center justify-between gap-3">
							<div class="flex items-center gap-3.5">
								<div class="bg-accent/10 flex h-10 w-10 items-center justify-center rounded-xl">
									<Sparkles class="text-accent h-5 w-5" />
								</div>
								<div>
									<h2 class="font-display text-lg leading-none font-bold tracking-tight">
										Configure Scan
									</h2>
									<p class="text-ink-muted mt-1 text-sm">
										Choose input, scanners, and run settings.
									</p>
								</div>
							</div>
							{#if hasValidInput && hasEnabledScanner && isAiConfigValid}
								<Chip tone="success" class="gap-1.5 font-medium">
									<CheckCircle2 class="h-3.5 w-3.5" />
									Ready to scan
								</Chip>
							{:else if isAiNavigatorEnabled && !isAiConfigValid}
								<Chip tone="warning" class="gap-1.5 font-medium">
									<AlertTriangle class="h-3.5 w-3.5" />
									Configure AI
								</Chip>
							{/if}
						</div>
					</div>

					<div class="space-y-7 p-6 lg:p-7">
						<div>
							<div class="mb-2 flex items-center gap-2">
								<span
									class="bg-accent/10 text-accent flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold"
									>1</span
								>
								<span class="form-section-label text-ink-muted">Input</span>
							</div>
							<PlaygroundModeToggle {mode} onModeChange={(m) => (mode = m)} />
						</div>

						<div>
							<div class="mb-2 flex items-center gap-2">
								<span
									class="bg-accent/10 text-accent flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold"
									>2</span
								>
								<span class="form-section-label text-ink-muted">Target</span>
							</div>
							{#if mode === 'url'}
								<PlaygroundUrlInput
									{urls}
									onUrlsChange={handleUrlsChange}
									onNormalize={normalizeUrlsIfNeeded}
								/>
							{/if}

							{#if mode === 'zip'}
								<PlaygroundZipUpload
									{file}
									onFileChange={handleFileChange}
									onError={handleFileError}
								/>
							{/if}
						</div>

						<div class="section-divider"></div>

						<div>
							<div class="mb-2 flex items-center gap-2">
								<span
									class="bg-accent/10 text-accent flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold"
									>3</span
								>
								<span class="form-section-label text-ink-muted">Scanners</span>
							</div>
							<PlaygroundScannerGrid
								{scanners}
								isLoading={isLoadingScanners}
								preset={scannerPreset}
								onPresetChange={handlePresetChange}
								onToggle={handleScannerToggle}
							/>
						</div>

						{#if isAiNavigatorEnabled}
							<PlaygroundAiConfig
								objective={aiObjective}
								model={aiModel}
								maxSteps={aiMaxSteps}
								maxWallTimeMs={aiMaxWallTimeMs}
								inputValues={aiInputValues}
								successCriteria={aiSuccessCriteria}
								isValid={isAiConfigValid}
								onObjectiveChange={(v) => (aiObjective = v)}
								onModelChange={(v) => (aiModel = v)}
								onMaxStepsChange={(v) => (aiMaxSteps = v)}
								onMaxWallTimeMsChange={(v) => (aiMaxWallTimeMs = v)}
								onInputValuesChange={(v) => (aiInputValues = v)}
								onSuccessCriteriaChange={(v) => (aiSuccessCriteria = v)}
							/>
						{/if}

						<div class="section-divider"></div>

						<div>
							<div class="mb-2 flex items-center gap-2">
								<span
									class="bg-accent/10 text-accent flex h-5 w-5 items-center justify-center rounded-full text-[10px] font-bold"
									>4</span
								>
								<span class="form-section-label text-ink-muted">Options</span>
							</div>
							<PlaygroundOptions
								{screenshot}
								{highlightStyle}
								onScreenshotChange={(v) => (screenshot = v)}
								onHighlightStyleChange={(v) => (highlightStyle = v)}
							/>
						</div>

						{#if error}
							<Alert variant="error">
								<div class="flex items-start gap-3">
									<AlertTriangle class="mt-0.5 h-5 w-5 shrink-0" />
									<div class="min-w-0">
										<p>{error}</p>
										{#if invalidUrls.length > 0}
											<ul class="mt-2 list-disc space-y-1 pl-5 text-sm">
												{#each invalidUrls.slice(0, 6) as item (item.url)}
													<li class="font-mono">{item.url} — {item.reason}</li>
												{/each}
												{#if invalidUrls.length > 6}
													<li class="text-ink-muted">...and {invalidUrls.length - 6} more</li>
												{/if}
											</ul>
										{/if}
									</div>
								</div>
							</Alert>
						{/if}

						<Button
							variant="glow"
							size="lg"
							class="h-12 w-full gap-2 rounded-xl text-base font-semibold"
							disabled={!canSubmit}
							onclick={handleSubmit}
						>
							{#if isSubmitting}
								<Loader2 class="h-5 w-5 animate-spin" />
								Starting Scan...
							{:else}
								<Play class="h-5 w-5" />
								Start Scan
							{/if}
						</Button>
					</div>
				</Panel>
			</div>

			<PlaygroundSidebar />
		</div>
	</div>
</PageSection>
