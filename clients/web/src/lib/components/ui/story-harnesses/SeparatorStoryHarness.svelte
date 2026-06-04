<script lang="ts">
	import Separator from '../Separator.svelte';

	interface Props {
		orientation?: 'horizontal' | 'vertical';
	}

	let { orientation = 'horizontal' }: Props = $props();
</script>

<div class={orientation === 'vertical' ? 'h-24' : 'w-full'}>
	<Separator {orientation} data-testid="separator" />
</div>
