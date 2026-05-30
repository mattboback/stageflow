<script lang="ts">
	import type { ScannerSelection } from '$lib/types/scan';

	import { goto } from '$app/navigation';
	import { fetchScanners, getDefaultScannerSelections, submitScanJob } from '$lib/api/client';
	import PlaygroundAiConfig from '$lib/components/playground/PlaygroundAiConfig.svelte';
	import PlaygroundAuthConfig from '$lib/components/playground/PlaygroundAuthConfig.svelte';
	import PlaygroundHeroSection from '$lib/components/playground/PlaygroundHeroSection.svelte';
	import PlaygroundModeToggle from '$lib/components/playground/PlaygroundModeToggle.svelte';
	import PlaygroundOptions from '$lib/components/playground/PlaygroundOptions.svelte';
	import PlaygroundScannerGrid from '$lib/components/playground/PlaygroundScannerGrid.svelte';
	import PlaygroundSidebar from '$lib/components/playground/PlaygroundSidebar.svelte';
	import PlaygroundUrlInput from '$lib/components/playground/PlaygroundUrlInput.svelte';
	import PlaygroundZipUpload from '$lib/components/playground/PlaygroundZipUpload.svelte';
	import {
		buildAiNavigatorConfig,
		buildFormAuthConfig,
		isAuthConfigComplete,
		normalizeUrlListText,
		parseUrlList,
		validateHttpUrls,
		type AuthFormConfig
	} from '$lib/components/playground/playground-utils';
	import {
		type ScannerPreset,
		applyScannerPreset,
		detectScannerPreset
	} from '$lib/domain/scanners/presets';
	import { Alert, Button, PageSection } from '$lib/components/ui';
	import { AlertTriangle, Loader2, Play } from 'lucide-svelte';
	import { slide } from 'svelte/transition';

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
	let scannerLoadError = $state<string | null>(null);
	let error = $state<string | null>(null);
	let invalidUrls = $state<Array<{ url: string; reason: string }>>([]);

	// AI Navigator configuration
	let aiObjective = $state('');
	let aiModel = $state(import.meta.env.VITE_AI_NAVIGATOR_DEFAULT_MODEL || 'openai/gpt-4o-mini');
	let aiMaxSteps = $state(10);
	let aiMaxWallTimeMs = $state(120000);
	let aiInputValues = $state<Array<{ key: string; value: string }>>([]);
	let aiSuccessCriteria = $state<Array<{ type: string; value: string }>>([]);

	// Auth configuration
	let authConfig = $state<AuthFormConfig>({
		enabled: false,
		loginUrl: '',
		username: '',
		password: '',
		usernameSelector: '',
		passwordSelector: '',
		submitSelector: '',
		successStrategy: 'selector',
		successSelector: ''
	});

	// Derived state
	const hasValidInput = $derived(mode === 'url' ? urls.trim().length > 0 : file !== null);
	const hasEnabledScanner = $derived(scanners.some((s) => s.enabled));
	const isAiNavigatorEnabled = $derived(scanners.some((s) => s.id === 'ai-navigator' && s.enabled));
	const enabledScannerCount = $derived(scanners.filter((s) => s.enabled).length);
	const isAiConfigValid = $derived(
		!isAiNavigatorEnabled || (aiObjective.trim().length > 0 && aiModel.trim().length > 0)
	);
	const isAuthEnabled = $derived(authConfig.enabled);
	const isAuthConfigValid = $derived(isAuthConfigComplete(authConfig));
	const canSubmit = $derived(
		hasValidInput &&
			hasEnabledScanner &&
			isAiConfigValid &&
			isAuthConfigValid &&
			!scannerLoadError &&
			!isSubmitting
	);
	const missingRequirements = $derived.by(() => {
		const requirements: string[] = [];
		if (!hasValidInput) {
			requirements.push(
				mode === 'url' ? 'Add at least one URL or switch to ZIP mode.' : 'Upload a ZIP archive.'
			);
		}
		if (scannerLoadError) {
			requirements.push('Scanner catalog failed to load. Refresh the page to retry.');
		} else if (!hasEnabledScanner) {
			requirements.push('Enable at least one scanner.');
		}
		if (isAiNavigatorEnabled && !isAiConfigValid) {
			requirements.push('Add an AI Navigator objective before running.');
		}
		if (mode === 'url' && isAuthEnabled && !isAuthConfigValid) {
			requirements.push(
				'Complete the authentication setup (login URL, username, password, and success selector required).'
			);
		}
		return requirements;
	});

	// Load scanners on mount
	$effect(() => {
		fetchScanners()
			.then((data) => {
				scannerLoadError = null;
				scanners = getDefaultScannerSelections(data.scanners);
				scannerPreset = detectScannerPreset(scanners);
			})
			.catch((e) => {
				const message = e instanceof Error ? e.message : 'Failed to load scanners. Refresh to retry.';
				scannerLoadError = message;
				error = message;
				scanners = [];
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

			const auth = mode === 'url' ? buildFormAuthConfig(authConfig) : null;

			const result = await submitScanJob({
				mode,
				urls: mode === 'url' ? valid : urlList,
				file,
				scanners,
				screenshot,
				highlightStyle,
				auth
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

<!-- Mobile sticky run bar -->
<div class="mobile-sticky-bar xl:hidden">
	<div class="flex flex-1 items-center gap-3">
		<div class="flex items-center gap-2">
			<span class="stat-mono text-ink-strong text-lg font-bold">{enabledScannerCount}</span>
			<span class="text-ink-muted text-xs">scanner{enabledScannerCount === 1 ? '' : 's'}</span>
		</div>
		{#if !canSubmit && missingRequirements.length > 0}
			<span class="text-ink-muted text-xs font-medium">
				{missingRequirements.length} step{missingRequirements.length === 1 ? '' : 's'} left
			</span>
		{/if}
	</div>
	<Button
		variant="default"
		size="lg"
		class="h-11 gap-2 rounded-xl px-6 text-sm font-semibold"
		disabled={!canSubmit}
		onclick={handleSubmit}
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

<PageSection
	class="playground-shell relative py-10 lg:py-14"
	padding="none"
	disableContainer={true}
>
	<div class="mx-auto max-w-7xl px-4 pb-24 sm:px-6 lg:px-8 xl:pb-0">
		<div
			class="grid items-start gap-8 xl:grid-cols-[minmax(0,1fr)_22rem] 2xl:grid-cols-[minmax(0,1fr)_24rem]"
		>
			<form
				class="border-line bg-surface rounded-2xl border shadow-[var(--shadow-sm)]"
				aria-label="Configure Scan"
				onsubmit={(e) => {
					e.preventDefault();
					handleSubmit();
				}}
			>
				<header
					class="border-line/80 flex flex-wrap items-center justify-between gap-3 border-b px-6 py-4 sm:px-8"
				>
					<div>
						<h2 class="text-ink text-base font-semibold tracking-[-0.005em]">Configure Scan</h2>
						<p class="text-ink-muted mt-0.5 text-xs">
							{enabledScannerCount} scanner{enabledScannerCount === 1 ? '' : 's'} selected
						</p>
					</div>
					<span
						class="rounded-full px-2.5 py-0.5 text-[11px] font-semibold"
						class:bg-emerald-50={canSubmit}
						class:text-emerald-700={canSubmit}
						class:bg-amber-50={!canSubmit}
						class:text-amber-700={!canSubmit}
					>
						{canSubmit
							? 'Ready'
							: `${missingRequirements.length} step${missingRequirements.length === 1 ? '' : 's'} left`}
					</span>
				</header>

				<div class="grid gap-8 px-6 py-7 sm:px-8 lg:grid-cols-[minmax(0,1fr)_minmax(20rem,0.85fr)]">
					<!-- Left: form steps -->
					<div class="space-y-7">
						<section aria-labelledby="step-1-title">
							<div class="form-step-head">
								<span class="form-step-num">1</span>
								<h3 id="step-1-title" class="form-step-title">Input</h3>
								<div class="form-step-rule" aria-hidden="true"></div>
							</div>
							<PlaygroundModeToggle {mode} onModeChange={(m) => (mode = m)} />
						</section>

						<section aria-labelledby="step-2-title">
							<div class="form-step-head">
								<span class="form-step-num">2</span>
								<h3 id="step-2-title" class="form-step-title">Target</h3>
								<div class="form-step-rule" aria-hidden="true"></div>
							</div>
							{#if mode === 'url'}
								<PlaygroundUrlInput
									{urls}
									onUrlsChange={handleUrlsChange}
									onNormalize={normalizeUrlsIfNeeded}
								/>
							{:else}
								<PlaygroundZipUpload
									{file}
									onFileChange={handleFileChange}
									onError={handleFileError}
								/>
							{/if}
						</section>

						<section aria-labelledby="step-3-title">
							<div class="form-step-head">
								<span class="form-step-num">3</span>
								<h3 id="step-3-title" class="form-step-title">Options</h3>
								<div class="form-step-rule" aria-hidden="true"></div>
							</div>
							<PlaygroundOptions
								{screenshot}
								{highlightStyle}
								onScreenshotChange={(v) => (screenshot = v)}
								onHighlightStyleChange={(v) => (highlightStyle = v)}
							/>
						</section>

						{#if isAiNavigatorEnabled}
							<section transition:slide={{ duration: 200 }} aria-labelledby="step-ai-title">
								<div class="form-step-head">
									<span class="form-step-num">A</span>
									<h3 id="step-ai-title" class="form-step-title">AI Navigator</h3>
									<div class="form-step-rule" aria-hidden="true"></div>
								</div>
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
							</section>
						{/if}

						{#if mode === 'url'}
							<section transition:slide={{ duration: 200 }} aria-labelledby="step-auth-title">
								<div class="form-step-head">
									<span class="form-step-num">B</span>
									<h3 id="step-auth-title" class="form-step-title">Authentication</h3>
									<div class="form-step-rule" aria-hidden="true"></div>
								</div>
								<PlaygroundAuthConfig
									config={authConfig}
									isValid={isAuthConfigValid}
									onConfigChange={(v) => (authConfig = v)}
								/>
							</section>
						{/if}

						{#if error}
							<Alert variant="error">
								<div class="flex items-start gap-3">
									<AlertTriangle class="mt-0.5 h-5 w-5 shrink-0" aria-hidden="true" />
									<div class="min-w-0">
										<p>{error}</p>
										{#if invalidUrls.length > 0}
											<ul class="mt-2 list-disc space-y-1 pl-5 text-sm">
												{#each invalidUrls.slice(0, 6) as item (item.url)}
													<li class="font-mono">{item.url} — {item.reason}</li>
												{/each}
												{#if invalidUrls.length > 6}
													<li class="text-ink-muted">…and {invalidUrls.length - 6} more</li>
												{/if}
											</ul>
										{/if}
									</div>
								</div>
							</Alert>
						{/if}
					</div>

					<!-- Right: scanner grid -->
					<aside aria-labelledby="step-4-title">
						<div class="form-step-head">
							<span class="form-step-num">4</span>
							<h3 id="step-4-title" class="form-step-title">Scanners</h3>
							<div class="form-step-rule" aria-hidden="true"></div>
						</div>
						<PlaygroundScannerGrid
							{scanners}
							isLoading={isLoadingScanners}
							loadError={scannerLoadError}
							preset={scannerPreset}
							onPresetChange={handlePresetChange}
							onToggle={handleScannerToggle}
						/>
					</aside>
				</div>

				<!-- Footer action bar (xl+ shows full version, smaller relies on sticky bar) -->
				<footer
					class="border-line/80 hidden flex-wrap items-center justify-between gap-4 border-t px-6 py-4 sm:px-8 lg:flex"
				>
					<div class="text-ink-muted min-w-0 flex-1 text-xs leading-relaxed">
						{#if canSubmit}
							{mode === 'url' ? 'URL target ready.' : 'ZIP upload ready.'}
							{enabledScannerCount} scanner{enabledScannerCount === 1 ? '' : 's'} selected. Screenshots
							{screenshot ? 'on' : 'off'}.
						{:else}
							<ul class="space-y-1">
								{#each missingRequirements as requirement (requirement)}
									<li class="flex items-start gap-2">
										<AlertTriangle
											class="mt-0.5 h-3.5 w-3.5 shrink-0 text-amber-600"
											aria-hidden="true"
										/>
										<span>{requirement}</span>
									</li>
								{/each}
							</ul>
						{/if}
					</div>
					<Button
						type="submit"
						variant="default"
						size="lg"
						class="h-11 gap-2 rounded-xl px-6 text-sm font-semibold"
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
				</footer>
			</form>

			<!-- Sticky run summary, xl+ only -->
			<div class="hidden xl:block">
				<PlaygroundSidebar
					{mode}
					{scanners}
					{screenshot}
					{highlightStyle}
					{canSubmit}
					{isAiNavigatorEnabled}
					{isAiConfigValid}
					{isAuthEnabled}
					{isAuthConfigValid}
					{missingRequirements}
				/>
			</div>
		</div>
	</div>
</PageSection>
