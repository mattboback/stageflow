<script lang="ts">
	import type { SelectSize } from '../select';

	import Select from '../Select.svelte';

	type SelectOption = {
		label: string;
		value: string;
	};

	interface Props {
		label?: string;
		options?: SelectOption[];
		value?: string;
		uiSize?: SelectSize;
		error?: boolean;
		disabled?: boolean;
		onChange?: (value: string) => void;
	}

	let {
		label = 'Scanner',
		options = [
			{ label: 'Axe', value: 'axe' },
			{ label: 'Lighthouse', value: 'lighthouse' },
			{ label: 'SEO', value: 'seo' }
		],
		value = $bindable('axe'),
		uiSize = 'md',
		error = false,
		disabled = false,
		onChange
	}: Props = $props();

	function handleChange(event: Event) {
		const currentTarget = event.currentTarget;
		if (!(currentTarget instanceof HTMLSelectElement)) return;
		onChange?.(currentTarget.value);
	}
</script>

<div class="w-[min(92vw,28rem)] space-y-2">
	<label class="text-sm font-medium" for="storybook-scanner-select">{label}</label>
	<Select
		id="storybook-scanner-select"
		data-testid="scanner-select"
		bind:value
		{uiSize}
		{error}
		{disabled}
		onchange={handleChange}
	>
		{#each options as option (option.value)}
			<option value={option.value}>{option.label}</option>
		{/each}
	</Select>
	<p class="text-ink-muted text-sm" data-testid="selected-value">Selected: {value}</p>
</div>
