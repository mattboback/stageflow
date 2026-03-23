import type { AgentGoal, AgentStep, PagePerception } from './types';

export function buildDecisionPrompt(
	perception: PagePerception,
	goal: AgentGoal,
	history: AgentStep[]
): string {
	const inputsAllowed = Object.keys(goal.inputValues ?? {});
	const inputPolicy =
		inputsAllowed.length > 0
			? `You may only fill/select using one of these valueKey values: ${inputsAllowed
					.slice(0, 20)
					.join(', ')}`
			: 'Do not use fill/select unless explicitly allowed (no input values provided).';

	const recent = history
		.slice(-5)
		.map((s) => `${s.stepNumber}. ${s.url} -> ${s.action.type}`)
		.join('\n');

	const elementList = perception.interactiveElements
		.slice(0, 25)
		.map((e, i) => `${i + 1}. ${e.selector} (${e.accessibleName ?? e.text ?? e.tagName})`)
		.join('\n');

	return [
		'You are a browser automation agent deciding the next Playwright action.',
		'',
		`Goal: ${goal.objective}`,
		inputPolicy,
		'',
		`Current URL: ${perception.url}`,
		`Title: ${perception.title}`,
		`Page type: ${perception.pageType}`,
		`Description: ${perception.description}`,
		'',
		'Recent steps:',
		recent || '(none)',
		'',
		'Interactive elements (selectors):',
		elementList || '(none)',
		'',
		'Return ONLY JSON with this shape:',
		'{',
		'  "action": {',
		'    "type": "click|fill|select|hover|scroll|keyboard|wait|done|stuck",',
		'    "selector": "required for click/fill/select/hover, optional for scroll",',
		'    "valueKey": "required for fill/select when inputs are allowed",',
		'    "direction": "up|down (scroll only)",',
		'    "pixels": 300,',
		'    "key": "Enter (keyboard only)",',
		'    "ms": 500 (wait only),',
		'    "reason": "stuck reason (stuck only)"',
		'  },',
		'  "reasoning": "short explanation",',
		'  "confidence": 0.0',
		'}'
	].join('\n');
}
