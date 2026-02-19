<script lang="ts">
	import PlaygroundModeToggle from '../PlaygroundModeToggle.svelte';

	type Mode = 'url' | 'zip';

	interface Props {
		initialMode?: Mode;
		onModeChange?: (mode: Mode) => void;
	}

	const { initialMode = 'url', onModeChange }: Props = $props();

	let mode = $derived<Mode>(initialMode);

	function handleModeChange(nextMode: Mode) {
		mode = nextMode;
		onModeChange?.(nextMode);
	}
</script>

<div class="space-y-4">
	<PlaygroundModeToggle mode={mode} onModeChange={handleModeChange} />
	<p class="text-ink-muted text-xs" data-testid="selected-mode">Selected mode: {mode}</p>
</div>
