<script lang="ts">
	import type { BadgeVariant } from '../badge';

	import Badge from '../Badge.svelte';

	interface Props {
		variant?: BadgeVariant;
		label?: string;
	}

	let { variant = 'default', label = 'Live' }: Props = $props();
</script>

<Badge {variant} data-testid="badge">
	{label}
</Badge>
