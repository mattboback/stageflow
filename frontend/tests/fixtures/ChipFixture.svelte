<script lang="ts">
	import { Chip } from '$lib/components/ui';
</script>

<div>
	<Chip data-testid="chip-span">Hello</Chip>
	<Chip as="button" type="button" data-testid="chip-button">Click</Chip>
</div>
