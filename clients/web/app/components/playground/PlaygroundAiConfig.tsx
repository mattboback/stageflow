import type {
	AiConfigState,
	AiInputValue,
	AiSuccessCriterion
} from '../../lib/components/playground/playground-utils';
import { DEFAULT_AI_CONFIG } from '../../lib/components/playground/playground-utils';
import { IS_HOSTED_DEMO } from '../../lib/site-metadata';

interface Props {
	config: AiConfigState;
	onConfigChange: (config: AiConfigState) => void;
}

export function PlaygroundAiConfig({ config, onConfigChange }: Props) {
	const update = (patch: Partial<AiConfigState>) => {
		onConfigChange({ ...config, ...patch });
	};

	const updateInputValue = (idx: number, patch: Partial<AiInputValue>) => {
		const next = config.inputValues.map((row, i) =>
			i === idx ? { ...row, ...patch } : row
		);
		update({ inputValues: next });
	};

	const addInputValue = () => {
		update({ inputValues: [...config.inputValues, { key: '', value: '' }] });
	};

	const removeInputValue = (idx: number) => {
		update({ inputValues: config.inputValues.filter((_, i) => i !== idx) });
	};

	const updateCriterion = (idx: number, patch: Partial<AiSuccessCriterion>) => {
		const next = config.successCriteria.map((row, i) =>
			i === idx ? { ...row, ...patch } : row
		);
		update({ successCriteria: next });
	};

	const addCriterion = () => {
		update({
			successCriteria: [...config.successCriteria, { type: 'url-contains', value: '' }]
		});
	};

	const removeCriterion = (idx: number) => {
		update({
			successCriteria: config.successCriteria.filter((_, i) => i !== idx)
		});
	};

	return (
		<section className="pai">
			<header className="pai__head">
				<h2 className="pai__title">AI Navigator</h2>
				<p className="pai__sub">
					Give the navigator a high-level goal; it drives the page to satisfy it.
				</p>
			</header>
			<div className="sensitive-data-warning" role="note">
				<strong>
					{IS_HOSTED_DEMO
						? 'Hosted demo: use synthetic test data only.'
						: 'Use synthetic or dedicated test data.'}
				</strong>{' '}
				Input values are sent to the navigator for this run. Never provide personal,
				production, payment, or reusable account data.
			</div>

			<label className="pai__field">
				<span className="label">
					Objective <span className="pai__req">*</span>
				</span>
				<textarea
					id="ai-objective"
					rows={3}
					placeholder="Add a Pro plan subscription and reach the checkout success page"
					value={config.objective}
					aria-required="true"
					aria-invalid={!config.objective.trim() ? true : undefined}
					onChange={(e) => update({ objective: e.target.value })}
				/>
			</label>

			<div className="pai__row">
				<label className="pai__field">
					<span className="label">Max steps</span>
					<input
						id="ai-max-steps"
						type="number"
						min={1}
						max={50}
						value={config.maxSteps}
						onChange={(e) =>
							update({ maxSteps: Number(e.target.value) || DEFAULT_AI_CONFIG.maxSteps })
						}
					/>
				</label>
				<label className="pai__field">
					<span className="label">Max wall time (ms)</span>
					<input
						id="ai-wall-time"
						type="number"
						min={10_000}
						max={600_000}
						step={10_000}
						value={config.maxWallTimeMs}
						onChange={(e) =>
							update({
								maxWallTimeMs:
									Number(e.target.value) || DEFAULT_AI_CONFIG.maxWallTimeMs
							})
						}
					/>
				</label>
				<label className="pai__field">
					<span className="label">Model</span>
					<input
						id="ai-model"
						type="text"
						value={config.model}
						onChange={(e) => update({ model: e.target.value })}
					/>
				</label>
			</div>

			<fieldset className="pai__group">
				<legend>Input values</legend>
				<p className="pai__hint">
					Key/value pairs the navigator can paste into forms (e.g. email + password).
				</p>
				{config.inputValues.map((row, idx) => (
					<div key={idx} className="pai__pair">
						<input
							id={`ai-input-key-${idx}`}
							type="text"
							placeholder="email"
							value={row.key}
							onChange={(e) => updateInputValue(idx, { key: e.target.value })}
						/>
						<input
							id={`ai-input-value-${idx}`}
							type="text"
							placeholder="value"
							value={row.value}
							onChange={(e) => updateInputValue(idx, { value: e.target.value })}
						/>
						<button
							type="button"
							className="pai__remove"
							onClick={() => removeInputValue(idx)}
							aria-label="Remove input"
						>
							×
						</button>
					</div>
				))}
				<button type="button" className="pai__add" onClick={addInputValue}>
					+ Add input
				</button>
			</fieldset>

			<fieldset className="pai__group">
				<legend>Success criteria</legend>
				<p className="pai__hint">
					Conditions that mark the goal as achieved.
				</p>
				{config.successCriteria.map((row, idx) => (
					<div key={idx} className="pai__pair">
						<select
							value={row.type}
							onChange={(e) => updateCriterion(idx, { type: e.target.value })}
						>
							<option value="url-contains">URL contains</option>
							<option value="url-equals">URL equals</option>
							<option value="text-contains">Page text contains</option>
							<option value="selector-visible">Selector visible</option>
						</select>
						<input
							id={`ai-criterion-value-${idx}`}
							type="text"
							placeholder="value"
							value={row.value}
							onChange={(e) => updateCriterion(idx, { value: e.target.value })}
						/>
						<button
							type="button"
							className="pai__remove"
							onClick={() => removeCriterion(idx)}
							aria-label="Remove criterion"
						>
							×
						</button>
					</div>
				))}
				<button type="button" className="pai__add" onClick={addCriterion}>
					+ Add criterion
				</button>
			</fieldset>
		</section>
	);
}
