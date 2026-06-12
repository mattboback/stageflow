<script lang="ts">
	import type { ScannerSelection } from '$lib/types/scan';

	import { goto } from '$app/navigation';
	import { fetchScanners, getDefaultScannerSelections, submitScanJob } from '$lib/api/client';
	import LauncherPanel from '$lib/components/playground/LauncherPanel.svelte';
	import PlaygroundAiConfig from '$lib/components/playground/PlaygroundAiConfig.svelte';
	import PlaygroundAuthConfig from '$lib/components/playground/PlaygroundAuthConfig.svelte';
	import PlaygroundOptions from '$lib/components/playground/PlaygroundOptions.svelte';
	import PlaygroundScannerGrid from '$lib/components/playground/PlaygroundScannerGrid.svelte';
	import ScanHistoryTable from '$lib/components/playground/ScanHistoryTable.svelte';
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
	import { Alert, Button } from '$lib/components/ui';
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
	let advancedOpen = $state(false);

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
		successStrategy: 'auto',
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
				const message =
					e instanceof Error ? e.message : 'Failed to load scanners. Refresh to retry.';
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

<!-- Mobile sticky run bar — only while the long advanced form is open -->
{#if advancedOpen}
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
			class="h-11 gap-2 px-6 text-sm font-semibold"
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
{/if}

<div class="bg-paper min-h-screen">
	<div class="container-width pt-24 pb-24 sm:pt-28 xl:pb-16">
		<!-- Page header -->
		<header class="max-w-3xl">
			<p class="section-kicker">Playground</p>
			<h1 class="h1-display mt-3">Run a scan</h1>
			<p class="text-ink-muted mt-4 max-w-2xl text-base leading-relaxed">
				Paste a URL and go — or upload a static-site ZIP. Eight scanners, one merged report, no
				account required.
			</p>
		</header>

		<form
			class="mt-8 max-w-4xl"
			aria-label="Run a scan"
			onsubmit={(e) => {
				e.preventDefault();
				handleSubmit();
			}}
		>
			<LauncherPanel
				{mode}
				{urls}
				{file}
				preset={scannerPreset}
				hasPresets={!isLoadingScanners && scanners.length > 1}
				{enabledScannerCount}
				{canSubmit}
				{isSubmitting}
				{missingRequirements}
				{advancedOpen}
				onModeChange={(m) => (mode = m)}
				onUrlsChange={handleUrlsChange}
				onNormalize={normalizeUrlsIfNeeded}
				onFileChange={handleFileChange}
				onFileError={handleFileError}
				onPresetChange={handlePresetChange}
				onToggleAdvanced={() => (advancedOpen = !advancedOpen)}
			/>

			{#if advancedOpen}
				<div
					id="advanced-options"
					transition:slide={{ duration: 200 }}
					class="border-line bg-surface mt-4 rounded-md border"
				>
					<div class="grid gap-8 p-4 sm:p-6 lg:grid-cols-2">
						<section aria-labelledby="advanced-scanners-title">
							<h3 id="advanced-scanners-title" class="sr-only">Scanners</h3>
							<PlaygroundScannerGrid
								{scanners}
								isLoading={isLoadingScanners}
								loadError={scannerLoadError}
								onToggle={handleScannerToggle}
							/>
						</section>

						<div class="space-y-7">
							<section aria-labelledby="advanced-options-title">
								<div class="form-step-head">
									<h3 id="advanced-options-title" class="form-step-title">Options</h3>
									<div class="form-step-rule" aria-hidden="true"></div>
								</div>
								<PlaygroundOptions
									{screenshot}
									{highlightStyle}
									onScreenshotChange={(v) => (screenshot = v)}
									onHighlightStyleChange={(v) => (highlightStyle = v)}
								/>
							</section>

							{#if mode === 'url'}
								<section transition:slide={{ duration: 200 }} aria-labelledby="advanced-auth-title">
									<div class="form-step-head">
										<h3 id="advanced-auth-title" class="form-step-title">Authentication</h3>
										<div class="form-step-rule" aria-hidden="true"></div>
									</div>
									<PlaygroundAuthConfig
										config={authConfig}
										isValid={isAuthConfigValid}
										onConfigChange={(v) => (authConfig = v)}
									/>
								</section>
							{/if}

							{#if isAiNavigatorEnabled}
								<section transition:slide={{ duration: 200 }} aria-labelledby="advanced-ai-title">
									<div class="form-step-head">
										<h3 id="advanced-ai-title" class="form-step-title">AI Navigator</h3>
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
						</div>
					</div>
				</div>
			{/if}

			{#if error}
				<div class="mt-4">
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
				</div>
			{/if}
		</form>

		<div class="mt-12 max-w-4xl">
			<ScanHistoryTable />
		</div>
	</div>
</div>
