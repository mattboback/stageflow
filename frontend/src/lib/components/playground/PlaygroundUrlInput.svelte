<script lang="ts">
	import { parseUrlList } from '$lib/components/playground/playground-utils';
	import { Label, Textarea } from '$lib/components/ui';

	interface Props {
		urls: string;
		onUrlsChange: (urls: string) => void;
		onNormalize: () => void;
	}

	const { urls, onUrlsChange, onNormalize }: Props = $props();

	const urlCount = $derived(parseUrlList(urls).length);
	const placeholder = ['example.com', 'example.com/about', 'https://example.com/contact'].join(
		'\n'
	);
</script>

<div class="animate-fade-in">
	<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
		<div class="flex items-center gap-2">
			<Label for="urls" class="text-sm font-semibold">URLs to Scan</Label>
		</div>
		<div class="flex items-center gap-3">
			{#if urlCount > 0}
				<span class="text-ink-muted text-xs font-medium">
					<span class="stat-mono text-accent">{urlCount}</span> / 100 URLs
				</span>
			{/if}
		</div>
	</div>
	<Textarea
		id="urls"
		value={urls}
		oninput={(e) => {
			onUrlsChange(e.currentTarget.value);
		}}
		onblur={onNormalize}
		placeholder={placeholder}
		rows={5}
		class="font-mono rounded-xl text-sm"
	/>
</div>
