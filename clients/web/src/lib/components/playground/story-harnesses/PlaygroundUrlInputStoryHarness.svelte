<script lang="ts">
	import PlaygroundUrlInput from '../PlaygroundUrlInput.svelte';

	interface Props {
		initialUrls?: string;
		onUrlsChange?: (urls: string) => void;
		onNormalize?: () => void;
	}

	let { initialUrls = '', onUrlsChange, onNormalize }: Props = $props();

	let urls = $derived(initialUrls);

	function handleUrlsChange(nextUrls: string) {
		urls = nextUrls;
		onUrlsChange?.(nextUrls);
	}

	function handleNormalize() {
		onNormalize?.();
	}
</script>

<PlaygroundUrlInput {urls} onUrlsChange={handleUrlsChange} onNormalize={handleNormalize} />
