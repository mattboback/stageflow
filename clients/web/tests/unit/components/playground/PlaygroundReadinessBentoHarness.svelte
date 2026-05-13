<script lang="ts">
	import PlaygroundReadinessBento from '$lib/components/playground/PlaygroundReadinessBento.svelte';
</script>

<PlaygroundReadinessBento
	navLabel="Configure Scan"
	readyLabel="Ready"
	isReady={true}
	scannerCount={3}
>
	{#snippet leftPanel()}
		<p>Left panel content</p>
	{/snippet}
	{#snippet rightPanel()}
		<p>Right panel content</p>
	{/snippet}
	{#snippet bottomLeft()}
		<p>Bottom left</p>
	{/snippet}
	{#snippet bottomCenter()}
		<p>Bottom center</p>
	{/snippet}
	{#snippet bottomRight()}
		<p>Bottom right</p>
	{/snippet}
</PlaygroundReadinessBento>
