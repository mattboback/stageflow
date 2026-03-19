<script lang="ts">
	import Input from '../Input.svelte';

	interface Props {
		label?: string;
		placeholder?: string;
		value?: string;
		error?: boolean;
		disabled?: boolean;
		onInput?: (value: string) => void;
	}

	let {
		label = 'Email',
		placeholder = 'name@example.com',
		value = $bindable(''),
		error = false,
		disabled = false,
		onInput
	}: Props = $props();

	function handleInput(event: Event) {
		const currentTarget = event.currentTarget;
		if (!(currentTarget instanceof HTMLInputElement)) return;
		onInput?.(currentTarget.value);
	}
</script>

<div class="w-[min(92vw,28rem)] space-y-2">
	<label class="text-sm font-medium" for="storybook-input">{label}</label>
	<Input
		id="storybook-input"
		data-testid="input"
		type="text"
		bind:value
		{placeholder}
		{error}
		{disabled}
		oninput={handleInput}
	/>
	<p class="text-sm text-ink-muted" data-testid="input-value">Value: {value}</p>
</div>
