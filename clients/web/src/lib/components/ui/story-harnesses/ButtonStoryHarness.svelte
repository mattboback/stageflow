<script lang="ts">
	import type { ButtonSize, ButtonVariant } from '../button';

	import Button from '../Button.svelte';

	interface Props {
		label?: string;
		variant?: ButtonVariant;
		size?: ButtonSize;
		disabled?: boolean;
		onClick?: () => void;
	}

	let {
		label = 'Run scan',
		variant = 'default',
		size = 'default',
		disabled = false,
		onClick
	}: Props = $props();

	function handleClick() {
		onClick?.();
	}
</script>

<Button {variant} {size} {disabled} type="button" onclick={handleClick}>
	{label}
</Button>
