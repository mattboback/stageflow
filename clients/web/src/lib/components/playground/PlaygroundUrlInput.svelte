<script lang="ts">
	import {
		normalizeUrlListText,
		parseUrlList,
		validateHttpUrls
	} from '$lib/components/playground/playground-utils';
	import { Label, Textarea } from '$lib/components/ui';
	import { AlertCircle, CheckCircle2, Info } from 'lucide-svelte';

	interface Props {
		urls: string;
		onUrlsChange: (urls: string) => void;
		onNormalize: () => void;
	}

	let { urls, onUrlsChange, onNormalize }: Props = $props();

	let hasInteracted = $state(false);
	let wasAutoNormalized = $state(false);

	const parsedUrls = $derived(parseUrlList(urls));
	const urlCount = $derived(parsedUrls.length);
	const validation = $derived(
		urlCount > 0 ? validateHttpUrls(parsedUrls) : { valid: [], invalid: [] }
	);
	const isAllValid = $derived(urlCount > 0 && validation.invalid.length === 0);
	const hasErrors = $derived(hasInteracted && validation.invalid.length > 0);

	const placeholder = ['https://example.com', 'example.com/pricing', 'example.com/contact'].join(
		'\n'
	);

	function handleBlur() {
		hasInteracted = true;
		const { changed } = normalizeUrlListText(urls);
		if (changed) {
			wasAutoNormalized = true;
			// Reset notice after a few seconds
			setTimeout(() => {
				wasAutoNormalized = false;
			}, 4000);
		}
		onNormalize();
	}

	function handleInput(e: Event & { currentTarget: HTMLTextAreaElement }) {
		hasInteracted = true;
		wasAutoNormalized = false;
		onUrlsChange(e.currentTarget.value);
	}
</script>

<div class="animate-fade-in">
	<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
		<div class="flex items-center gap-2">
			<Label for="urls" class="text-sm font-bold">URLs to Scan</Label>
			{#if isAllValid}
				<CheckCircle2 class="h-4 w-4 text-green-500" />
			{/if}
		</div>
		<div class="flex items-center gap-3">
			{#if urlCount > 0}
				<span class="text-ink-muted text-xs font-medium">
					<span class="stat-mono {hasErrors ? 'text-red-500' : 'text-accent'}">{urlCount}</span> / 100
					URLs
				</span>
			{/if}
		</div>
	</div>
	<Textarea
		id="urls"
		value={urls}
		oninput={handleInput}
		onblur={handleBlur}
		{placeholder}
		rows={5}
		error={hasErrors}
		class="rounded-xl font-mono text-sm"
	/>

	<p class="text-ink-muted mt-2 text-xs leading-relaxed">
		Enter one URL per line. Bare domains are okay and will be normalized to `https://` when
		possible.
	</p>

	{#if hasErrors}
		<div class="mt-2 flex items-start gap-1.5 text-xs text-red-500">
			<AlertCircle class="mt-0.5 h-3.5 w-3.5 shrink-0" />
			<span>
				Fix {validation.invalid.length} invalid
				{validation.invalid.length === 1 ? 'URL' : 'URLs'} to continue. First error:
				{validation.invalid[0].reason}
			</span>
		</div>
	{:else if wasAutoNormalized}
		<div class="text-ink mt-2 flex items-start gap-1.5 text-xs">
			<Info class="mt-0.5 h-3.5 w-3.5 shrink-0" />
			<span>We formatted your entries automatically, including adding `https://` where needed.</span
			>
		</div>
	{:else if urlCount > 0 && !hasErrors}
		<div class="animate-fade-in mt-2 flex items-start gap-1.5 text-xs text-green-700">
			<CheckCircle2 class="mt-0.5 h-3.5 w-3.5 shrink-0" />
			<span>All {urlCount} {urlCount === 1 ? 'target is ready' : 'targets are ready'}.</span>
		</div>
	{/if}
</div>
