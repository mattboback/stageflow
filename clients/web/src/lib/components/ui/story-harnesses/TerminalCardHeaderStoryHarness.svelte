<script lang="ts">
	import TerminalCardHeader from '../TerminalCardHeader.svelte';

	interface Props {
		path: string;
		showBadges?: boolean;
	}

	let { path = 'stageflow scan https://example.com', showBadges = true }: Props = $props();

	const badges = $derived(
		showBadges ? [{ label: 'live' }, { label: 'json', variant: 'terminal' as const }] : []
	);
</script>

<div class="border-line overflow-hidden rounded border" data-testid="terminal-header-shell">
	<TerminalCardHeader {path} {badges} data-testid="terminal-header" />
</div>
