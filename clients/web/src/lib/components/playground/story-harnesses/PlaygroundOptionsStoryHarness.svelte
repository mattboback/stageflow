<script lang="ts">
	import type { BrowserEngine } from '$lib/api/client';

	import PlaygroundOptions from '../PlaygroundOptions.svelte';

	interface Props {
		initialScreenshot?: boolean;
		initialHighlightStyle?: 'solid' | 'dashed';
		initialEngine?: BrowserEngine;
		onScreenshotChange?: (value: boolean) => void;
		onHighlightStyleChange?: (value: 'solid' | 'dashed') => void;
		onEngineChange?: (value: BrowserEngine) => void;
	}

	let {
		initialScreenshot = true,
		initialHighlightStyle = 'solid',
		initialEngine = 'chromium',
		onScreenshotChange,
		onHighlightStyleChange,
		onEngineChange
	}: Props = $props();

	let screenshot = $state(initialScreenshot);
	let highlightStyle = $state<'solid' | 'dashed'>(initialHighlightStyle);
	let engine = $state<BrowserEngine>(initialEngine);

	function handleScreenshotChange(value: boolean) {
		screenshot = value;
		onScreenshotChange?.(value);
	}

	function handleHighlightStyleChange(value: 'solid' | 'dashed') {
		highlightStyle = value;
		onHighlightStyleChange?.(value);
	}

	function handleEngineChange(value: BrowserEngine) {
		engine = value;
		onEngineChange?.(value);
	}
</script>

<PlaygroundOptions
	{screenshot}
	{highlightStyle}
	{engine}
	onScreenshotChange={handleScreenshotChange}
	onHighlightStyleChange={handleHighlightStyleChange}
	onEngineChange={handleEngineChange}
/>
