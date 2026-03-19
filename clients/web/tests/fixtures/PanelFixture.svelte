<script lang="ts">
	import Panel from '$lib/components/ui/Panel.svelte';

	type Props = {
		variant?: 'default' | 'muted' | 'ghost';
		padding?: 'none' | 'xs' | 'sm' | 'md' | 'lg' | 'xl';
		rounded?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl';
		interactive?: boolean;
		class?: string;
	};

	const { variant, padding, rounded, interactive, class: className }: Props = $props();
</script>

<Panel {variant} {padding} {rounded} {interactive} class={className} data-testid="panel">
	<div data-testid="panel-child">Panel content</div>
</Panel>
