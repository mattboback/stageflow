<script lang="ts">
	import type { ScannerSelection } from '$lib/types/scan';

	import { goto } from '$app/navigation';
	import { fetchScanners, getDefaultScannerSelections, submitScanJob } from '$lib/api/client';
	import {
		buildAiNavigatorConfig,
		normalizeUrlListText,
		parseUrlList,
		validateHttpUrls
	} from '$lib/components/playground/playground-utils';
	import PlaygroundAiConfig from '$lib/components/playground/PlaygroundAiConfig.svelte';
	import PlaygroundHeroSection from '$lib/components/playground/PlaygroundHeroSection.svelte';
	import PlaygroundModeToggle from '$lib/components/playground/PlaygroundModeToggle.svelte';
	import PlaygroundOptions from '$lib/components/playground/PlaygroundOptions.svelte';
	import PlaygroundScannerGrid from '$lib/components/playground/PlaygroundScannerGrid.svelte';
	import PlaygroundSidebar from '$lib/components/playground/PlaygroundSidebar.svelte';
	import PlaygroundUrlInput from '$lib/components/playground/PlaygroundUrlInput.svelte';
	import PlaygroundZipUpload from '$lib/components/playground/PlaygroundZipUpload.svelte';
	import {
		applyScannerPreset,
		detectScannerPreset,
		type ScannerPreset
	} from '$lib/components/playground/scanner-presets';
	import { Alert, Button, Chip, PageSection, Panel } from '$lib/components/ui';
	import { AlertTriangle, CheckCircle2, Info, Loader2, Play, ScanSearch } from 'lucide-svelte';

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
	let info = $state<string | null>(null);
	let invalidUrls = $state<Array<{ url: string; reason: string }>>([]);

	// AI Navigator configuration
	let aiObjective = $state('');
	let aiModel = $state('openai/gpt-4o-mini');
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
		info = "Added 'https://' to URLs that didn't include a scheme.";
	}

	function handleUrlsChange(newUrls: string) {
		urls = newUrls;
		info = null;
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
		info = null;
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

<PageSection class="py-12">
	<div class="container-width">
		<div class="grid gap-8 lg:grid-cols-3">
			<!-- Main Form -->
			<div class="lg:col-span-2">
				<Panel padding="none" rounded="xl" class="text-ink overflow-hidden">
					<!-- Header -->
					<div class="border-line border-b p-5">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-3">
								<div class="bg-accent/10 flex h-9 w-9 items-center justify-center rounded-lg">
									<ScanSearch class="text-accent h-[18px] w-[18px]" />
								</div>
								<div>
									<h3 class="text-base leading-none font-semibold tracking-tight">
										Configure Scan
									</h3>
									<p class="text-ink-muted mt-0.5 text-xs">Select input and scanners</p>
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

					<!-- Form Content -->
					<div class="space-y-8 p-6 pt-6">
						<!-- Mode Toggle -->
						<PlaygroundModeToggle {mode} onModeChange={(m) => (mode = m)} />

						<!-- URL Input -->
						{#if mode === 'url'}
							<PlaygroundUrlInput
								{urls}
								onUrlsChange={handleUrlsChange}
								onNormalize={normalizeUrlsIfNeeded}
							/>
						{/if}

						<!-- ZIP Upload -->
						{#if mode === 'zip'}
							<PlaygroundZipUpload
								{file}
								onFileChange={handleFileChange}
								onError={handleFileError}
							/>
						{/if}

						<!-- Scanner Selection -->
						<PlaygroundScannerGrid
							{scanners}
							isLoading={isLoadingScanners}
							preset={scannerPreset}
							onPresetChange={handlePresetChange}
							onToggle={handleScannerToggle}
						/>

						<!-- AI Navigator Configuration -->
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

						<!-- Options -->
						<PlaygroundOptions
							{screenshot}
							{highlightStyle}
							onScreenshotChange={(v) => (screenshot = v)}
							onHighlightStyleChange={(v) => (highlightStyle = v)}
						/>

						<!-- Info Display -->
						{#if info}
							<Alert variant="info">
								<div class="flex items-start gap-3">
									<Info class="mt-0.5 h-5 w-5 shrink-0" />
									<p>{info}</p>
								</div>
							</Alert>
						{/if}

						<!-- Error Display -->
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

						<!-- Submit Button -->
						<Button
							variant="glow"
							size="lg"
							class="w-full gap-2 text-base font-semibold"
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

			<!-- Sidebar Info -->
			<PlaygroundSidebar />
		</div>
	</div>
</PageSection>
