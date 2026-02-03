<script lang="ts">
	import PageSection from '$lib/components/ui/PageSection.svelte';

	interface Props {
		padding?: 'default' | 'page' | 'none';
		disableContainer?: boolean;
	}

	const { padding = 'default', disableContainer = false }: Props = $props();
</script>

<PageSection {padding} {disableContainer} data-testid="page-section">
	<div data-testid="page-section-child">Hello</div>
</PageSection>
