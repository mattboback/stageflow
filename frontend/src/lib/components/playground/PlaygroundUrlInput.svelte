<script lang="ts">
	import { parseUrlList } from '$lib/components/playground/playground-utils';
	import { Button, Chip, Label, Textarea } from '$lib/components/ui';

	interface Props {
		urls: string;
		onUrlsChange: (urls: string) => void;
		onNormalize: () => void;
	}

	const { urls, onUrlsChange, onNormalize }: Props = $props();

	const urlCount = $derived(parseUrlList(urls).length);
</script>

<div class="animate-fade-in">
	<div class="mb-2 flex flex-wrap items-center justify-between gap-2">
		<div class="flex items-center gap-2">
			<Label for="urls" class="text-sm font-semibold">URLs to Scan</Label>
			<Chip tone="default" size="xs" class="font-mono">auto-https</Chip>
		</div>
		<div class="flex items-center gap-3">
			{#if urlCount > 0}
				<span class="text-ink-muted text-xs font-medium">
					<span class="stat-mono text-accent">{urlCount}</span> / 100 URLs
				</span>
			{/if}
			<Button variant="ghost" size="sm" class="h-7 text-xs" onclick={onNormalize}>
				Format URLs
			</Button>
		</div>
	</div>
	<Textarea
		id="urls"
		value={urls}
		oninput={(e) => {
			onUrlsChange(e.currentTarget.value);
		}}
		onblur={onNormalize}
		placeholder="example.com&#10;example.com/about&#10;https://example.com/contact"
		rows={5}
		class="font-mono text-sm"
	/>
	<p class="text-ink-faint mt-2 flex items-center gap-1.5 text-xs">
		<span class="bg-ink-faint inline-block h-1 w-1 rounded-full"></span>
		Enter one URL per line. If you omit a scheme, Stageflow assumes
		<span class="font-mono">https://</span>.
	</p>
</div>
