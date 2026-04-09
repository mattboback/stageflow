<script lang="ts">
	import { Button, Chip, Label, Select, SelectField, Textarea } from '$lib/components/ui';
	import { cn } from '$lib/utils';
	import {
		AlertTriangle,
		Bot,
		ChevronDown,
		ChevronUp,
		Info,
		Plus,
		Sparkles,
		Trash2
	} from 'lucide-svelte';

	interface Props {
		objective: string;
		model: string;
		maxSteps: number;
		maxWallTimeMs: number;
		inputValues: Array<{ key: string; value: string }>;
		successCriteria: Array<{ type: string; value: string }>;
		isValid: boolean;
		onObjectiveChange: (value: string) => void;
		onModelChange: (value: string) => void;
		onMaxStepsChange: (value: number) => void;
		onMaxWallTimeMsChange: (value: number) => void;
		onInputValuesChange: (values: Array<{ key: string; value: string }>) => void;
		onSuccessCriteriaChange: (criteria: Array<{ type: string; value: string }>) => void;
	}

	let {
		objective,
		model,
		maxSteps,
		maxWallTimeMs,
		inputValues,
		successCriteria,
		isValid,
		onObjectiveChange,
		onModelChange,
		onMaxStepsChange,
		onMaxWallTimeMsChange,
		onInputValuesChange,
		onSuccessCriteriaChange
	}: Props = $props();

	let showAdvancedSettings = $state(false);

	// Available AI models
	const aiModels = [
		{ value: 'openai/gpt-4o-mini', label: 'GPT-4o Mini (Fast & Affordable)' },
		{ value: 'openai/gpt-4o', label: 'GPT-4o (Best Quality)' },
		{ value: 'anthropic/claude-3.5-sonnet', label: 'Claude 3.5 Sonnet' },
		{ value: 'anthropic/claude-3-haiku', label: 'Claude 3 Haiku (Fast)' },
		{ value: 'google/gemini-pro-vision', label: 'Gemini Pro Vision' }
	];

	// Success criteria types
	const successCriteriaTypes = [
		{ value: 'url-contains', label: 'URL Contains' },
		{ value: 'url-matches', label: 'URL Matches (Regex)' },
		{ value: 'element-visible', label: 'Element Visible (Selector)' },
		{ value: 'text-visible', label: 'Text Visible' },
		{ value: 'custom', label: 'Custom Condition' }
	];

	function addInputValue() {
		onInputValuesChange([...inputValues, { key: '', value: '' }]);
	}

	function removeInputValue(index: number) {
		onInputValuesChange(inputValues.filter((_, i) => i !== index));
	}

	function updateInputValue(index: number, field: 'key' | 'value', newValue: string) {
		onInputValuesChange(
			inputValues.map((item, i) => (i === index ? { ...item, [field]: newValue } : item))
		);
	}

	function addSuccessCriterion() {
		onSuccessCriteriaChange([...successCriteria, { type: 'url-contains', value: '' }]);
	}

	function removeSuccessCriterion(index: number) {
		onSuccessCriteriaChange(successCriteria.filter((_, i) => i !== index));
	}

	function updateSuccessCriterion(index: number, field: 'type' | 'value', newValue: string) {
		onSuccessCriteriaChange(
			successCriteria.map((item, i) => (i === index ? { ...item, [field]: newValue } : item))
		);
	}
</script>

<div
	class="animate-fade-in border-line/80 bg-surface-muted/55 space-y-5 rounded-2xl border p-6 shadow-[var(--shadow-xs)]"
>
	<div class="mb-4 flex items-center gap-3">
		<div class="bg-accent/10 flex h-10 w-10 items-center justify-center rounded-xl">
			<Bot class="text-accent h-5 w-5" />
		</div>
		<div>
			<h3 class="text-ink text-sm font-semibold">AI Navigator Configuration</h3>
			<p class="text-ink-muted text-xs">Configure how the AI agent will navigate your site</p>
		</div>
		{#if !isValid}
			<Chip tone="warning" size="xs" class="ml-auto gap-1 font-medium">
				<AlertTriangle class="h-3 w-3" />
				Required
			</Chip>
		{/if}
	</div>

	<!-- Objective (Required) -->
	<div>
		<Label for="ai-objective" class="mb-2 flex items-center gap-1 text-sm font-semibold">
			Goal Objective
			<span class="text-accent">*</span>
		</Label>
		<Textarea
			id="ai-objective"
			value={objective}
			oninput={(e) => onObjectiveChange(e.currentTarget.value)}
			placeholder="Navigate to the contact page and fill out the inquiry form with the provided test data…"
			name="aiObjective"
			rows={3}
			class={cn(
				'rounded-xl text-sm',
				!objective.trim() && 'border-amber-300 focus:border-amber-500'
			)}
		/>
		<p class="text-ink-faint mt-1.5 flex items-start gap-1.5 text-xs">
			<Info class="mt-0.5 h-3 w-3 shrink-0" />
			Describe what the AI agent should accomplish on your website.
		</p>
	</div>

	<!-- Model Selection (Required) -->
	<div>
		<Label for="ai-model" class="mb-2 flex items-center gap-1 text-sm font-semibold">
			AI Model
			<span class="text-accent">*</span>
		</Label>
		<SelectField
			id="ai-model"
			variant="prominent"
			value={model}
			onchange={(e) => onModelChange(e.currentTarget.value)}
			name="aiModel"
		>
			{#each aiModels as aiModel (aiModel.value)}
				<option value={aiModel.value}>{aiModel.label}</option>
			{/each}
		</SelectField>
	</div>

	<!-- Input Values (for form fills) -->
	<div>
		<div class="mb-2 flex items-center justify-between">
			<Label class="text-sm font-semibold">Input Values</Label>
			<Button variant="ghost" size="sm" class="h-7 gap-1 text-xs" onclick={addInputValue}>
				<Plus class="h-3 w-3" />
				Add Field
			</Button>
		</div>
		<p class="text-ink-faint mb-3 text-xs">
			Key-value pairs the AI can use to fill forms (e.g., email, name, message).
		</p>
		{#if inputValues.length === 0}
			<div class="bg-surface border-line rounded-xl border py-4 text-center">
				<p class="text-ink-muted text-xs">No input values defined</p>
			</div>
		{:else}
			<div class="space-y-2">
				{#each inputValues as inputValue, index (index)}
					<div class="flex items-start gap-2">
						<div class="flex-1">
							<Label for={`ai-input-key-${index}`} class="sr-only">
								Input field name {index + 1}
							</Label>
							<input
								id={`ai-input-key-${index}`}
								type="text"
								name={`aiInputValues[${index}].key`}
								autocomplete="off"
								placeholder="Field name…"
								value={inputValue.key}
								oninput={(e) => updateInputValue(index, 'key', e.currentTarget.value)}
								class="border-line bg-surface text-ink placeholder:text-ink-faint focus:border-accent flex-1 rounded-xl border px-3 py-2 text-sm focus:outline-none"
							/>
						</div>
						<div class="flex-1">
							<Label for={`ai-input-value-${index}`} class="sr-only">
								Input field value {index + 1}
							</Label>
							<input
								id={`ai-input-value-${index}`}
								type="text"
								name={`aiInputValues[${index}].value`}
								autocomplete="off"
								placeholder="Value…"
								value={inputValue.value}
								oninput={(e) => updateInputValue(index, 'value', e.currentTarget.value)}
								class="border-line bg-surface text-ink placeholder:text-ink-faint focus:border-accent flex-1 rounded-xl border px-3 py-2 text-sm focus:outline-none"
							/>
						</div>
						<button
							type="button"
							onclick={() => removeInputValue(index)}
							class="text-ink-muted flex h-9 w-9 shrink-0 items-center justify-center rounded-lg transition-colors hover:bg-red-50 hover:text-red-500"
							aria-label={`Remove input field ${index + 1}`}
						>
							<Trash2 class="h-4 w-4" />
						</button>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Advanced Settings Toggle -->
	<button
		type="button"
		onclick={() => (showAdvancedSettings = !showAdvancedSettings)}
		class="border-line bg-surface hover:bg-surface-muted flex w-full items-center justify-between rounded-xl border px-4 py-3 text-left transition-colors"
		aria-expanded={showAdvancedSettings}
		aria-controls="ai-advanced-settings"
	>
		<span class="text-ink text-sm font-medium">Advanced Settings</span>
		{#if showAdvancedSettings}
			<ChevronUp class="text-ink-muted h-4 w-4" />
		{:else}
			<ChevronDown class="text-ink-muted h-4 w-4" />
		{/if}
	</button>

	{#if showAdvancedSettings}
		<div id="ai-advanced-settings" class="animate-fade-in space-y-5 pt-2">
			<!-- Max Steps & Timeout -->
			<div class="grid gap-4 sm:grid-cols-2">
				<div>
					<Label for="ai-max-steps" class="mb-2 block text-sm font-semibold">Max Steps</Label>
					<input
						id="ai-max-steps"
						type="number"
						name="aiMaxSteps"
						min="1"
						max="50"
						value={maxSteps}
						oninput={(e) => onMaxStepsChange(parseInt(e.currentTarget.value) || 10)}
						class="border-line bg-surface text-ink focus:border-accent w-full rounded-xl border px-3 py-2.5 text-sm focus:outline-none"
					/>
					<p class="text-ink-faint mt-1 text-xs">Maximum navigation actions</p>
				</div>
				<div>
					<Label for="ai-timeout" class="mb-2 block text-sm font-semibold">Timeout (seconds)</Label>
					<input
						id="ai-timeout"
						type="number"
						name="aiTimeoutSeconds"
						min="10"
						max="300"
						value={maxWallTimeMs / 1000}
						oninput={(e) => onMaxWallTimeMsChange(parseInt(e.currentTarget.value) * 1000 || 120000)}
						class="border-line bg-surface text-ink focus:border-accent w-full rounded-xl border px-3 py-2.5 text-sm focus:outline-none"
					/>
					<p class="text-ink-faint mt-1 text-xs">Maximum wall clock time</p>
				</div>
			</div>

			<!-- Success Criteria -->
			<div>
				<div class="mb-2 flex items-center justify-between">
					<Label class="text-sm font-semibold">Success Criteria</Label>
					<Button variant="ghost" size="sm" class="h-7 gap-1 text-xs" onclick={addSuccessCriterion}>
						<Plus class="h-3 w-3" />
						Add Criterion
					</Button>
				</div>
				<p class="text-ink-faint mb-3 text-xs">Conditions to verify the goal was achieved.</p>
				{#if successCriteria.length === 0}
					<div class="bg-surface border-line rounded-xl border py-4 text-center">
						<p class="text-ink-muted text-xs">
							No success criteria defined (AI will determine completion)
						</p>
					</div>
				{:else}
					<div class="space-y-2">
						{#each successCriteria as criterion, index (index)}
							<div class="flex items-start gap-2">
								<Select
									aria-label={`Success criterion type ${index + 1}`}
									value={criterion.type}
									onchange={(e) => updateSuccessCriterion(index, 'type', e.currentTarget.value)}
									class="w-40 shrink-0"
									name={`aiSuccessCriteria[${index}].type`}
								>
									{#each successCriteriaTypes as type (type.value)}
										<option value={type.value}>{type.label}</option>
									{/each}
								</Select>
								<div class="flex-1">
									<Label for={`ai-success-criterion-${index}`} class="sr-only">
										Success criterion value {index + 1}
									</Label>
									<input
										id={`ai-success-criterion-${index}`}
										type="text"
										name={`aiSuccessCriteria[${index}].value`}
										autocomplete="off"
										placeholder={criterion.type === 'element-visible'
											? 'CSS selector…'
											: 'Value to match…'}
										value={criterion.value}
										oninput={(e) => updateSuccessCriterion(index, 'value', e.currentTarget.value)}
										class="border-line bg-surface text-ink placeholder:text-ink-faint focus:border-accent flex-1 rounded-xl border px-3 py-2 text-sm focus:outline-none"
									/>
								</div>
								<button
									type="button"
									onclick={() => removeSuccessCriterion(index)}
									class="text-ink-muted flex h-9 w-9 shrink-0 items-center justify-center rounded-lg transition-colors hover:bg-red-50 hover:text-red-500"
									aria-label={`Remove success criterion ${index + 1}`}
								>
									<Trash2 class="h-4 w-4" />
								</button>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		</div>
	{/if}

	<!-- AI Beta Notice -->
	<div class="bg-accent-soft/60 flex items-start gap-2 rounded-xl p-3">
		<Sparkles class="text-accent mt-0.5 h-4 w-4 shrink-0" />
		<p class="text-xs text-[color:var(--color-accent-ink)]">
			<strong>Experimental:</strong> AI Navigator uses vision models to understand and interact with your
			site. Results may vary based on page complexity.
		</p>
	</div>
</div>
