<script lang="ts">
import SelectField from "../SelectField.svelte";

type SelectFieldVariant = "default" | "prominent";

interface Props {
	label?: string;
	value?: string;
	variant?: SelectFieldVariant;
	onChange?: (value: string) => void;
}

let {
	label = "Scanner",
	value = $bindable("axe"),
	variant = "default",
	onChange,
}: Props = $props();

function handleChange(event: Event) {
	const currentTarget = event.currentTarget;
	if (!(currentTarget instanceof HTMLSelectElement)) return;
	onChange?.(currentTarget.value);
}
</script>

<div class="w-[min(92vw,28rem)] space-y-2">
	<label class="text-sm font-medium" for="storybook-select-field">{label}</label>
	<SelectField
		id="storybook-select-field"
		data-testid="select-field"
		bind:value
		{variant}
		onchange={handleChange}
	>
		<option value="axe">Axe</option>
		<option value="lighthouse">Lighthouse</option>
		<option value="seo">SEO</option>
	</SelectField>
	<p class="text-sm text-ink-muted" data-testid="select-field-value">Selected: {value}</p>
</div>
