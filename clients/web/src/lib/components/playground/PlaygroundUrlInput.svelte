<script lang="ts">
	import { normalizeUrlListText, parseUrlList, validateHttpUrls } from '$lib/components/playground/playground-utils';
	import { Label, Textarea } from '$lib/components/ui';
	import { CheckCircle2, AlertCircle, Info } from 'lucide-svelte';

	interface Props {
		urls: string;
		onUrlsChange: (urls: string) => void;
		onNormalize: () => void;
	}

	const { urls, onUrlsChange, onNormalize }: Props = $props();

	let hasInteracted = $state(false);
	let wasAutoNormalized = $state(false);

	const parsedUrls = $derived(parseUrlList(urls));
	const urlCount = $derived(parsedUrls.length);
	const validation = $derived(urlCount > 0 ? validateHttpUrls(parsedUrls) : { valid: [], invalid: [] });
	const isAllValid = $derived(urlCount > 0 && validation.invalid.length === 0);
	const hasErrors = $derived(hasInteracted && validation.invalid.length > 0);

	const placeholder = ['example.com', 'example.com/about', 'https://example.com/contact'].join(
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
				<CheckCircle2 class="text-green-500 h-4 w-4" />
			{/if}
		</div>
		<div class="flex items-center gap-3">
			{#if urlCount > 0}
				<span class="text-ink-muted text-xs font-medium">
					<span class="stat-mono {hasErrors ? 'text-red-500' : 'text-accent'}">{urlCount}</span> / 100 URLs
				</span>
			{/if}
		</div>
	</div>
	<Textarea
		id="urls"
		value={urls}
		oninput={handleInput}
		onblur={handleBlur}
		placeholder={placeholder}
		rows={5}
		error={hasErrors}
		class="font-mono rounded-xl text-sm"
	/>
	
	{#if hasErrors}
		<div class="mt-2 flex items-start gap-1.5 text-xs text-red-500">
			<AlertCircle class="mt-0.5 h-3.5 w-3.5 shrink-0" />
			<span>
				{validation.invalid.length} invalid {validation.invalid.length === 1 ? 'URL' : 'URLs'}. 
				First error: {validation.invalid[0].reason}
			</span>
		</div>
	{:else if wasAutoNormalized}
		<div class="mt-2 flex items-start gap-1.5 text-xs text-blue-500 animate-fade-in">
			<Info class="mt-0.5 h-3.5 w-3.5 shrink-0" />
			<span>URLs were automatically formatted (e.g. adding https://).</span>
		</div>
	{:else if urlCount > 0 && !hasErrors}
		<div class="mt-2 flex items-start gap-1.5 text-xs text-green-600 animate-fade-in">
			<CheckCircle2 class="mt-0.5 h-3.5 w-3.5 shrink-0" />
			<span>All {urlCount} {urlCount === 1 ? 'URL is' : 'URLs are'} valid.</span>
		</div>
	{/if}
</div>
