import { describe, expect, it } from 'vitest';

import { parseActionDecision } from '../../src/ai/action-decision-parser';

describe('parseActionDecision', () => {
	it('parses click actions', () => {
		const decision = parseActionDecision(
			`{"action":{"type":"click","selector":" #btn "},"reasoning":"click it","confidence":0.9}`,
			{ objective: 'Click the button' }
		);

		expect(decision).toEqual({
			action: { type: 'click', selector: '#btn' },
			reasoning: 'click it',
			confidence: 0.9
		});
	});

	it('rejects unsafe model-provided selectors', () => {
		for (const selector of [
			'//button',
			'xpath=//button',
			'javascript:alert(1)',
			'data:text/html,hello',
			'file:///etc/passwd',
			'http://example.com',
			'#login\nbutton',
			`#${'x'.repeat(513)}`
		]) {
			expect(
				parseActionDecision(
					JSON.stringify({
						action: { type: 'click', selector }
					}),
					{ objective: 'Click the button' }
				)
			).toBeUndefined();
		}
	});

	it('resolves fill/select valueKey against goal.inputValues', () => {
		const decision = parseActionDecision(
			`{"action":{"type":"fill","selector":"#email","valueKey":"email"}}`,
			{ objective: 'Fill form', inputValues: { email: 'me@example.com' } }
		);

		expect(decision?.action).toEqual({
			type: 'fill',
			selector: '#email',
			value: 'me@example.com'
		});
		expect(decision?.confidence).toBe(0.7);
		expect(decision?.reasoning).toBe('');
	});

	it('returns a stuck action when fill/select is disallowed', () => {
		const decision = parseActionDecision(
			`{"action":{"type":"fill","selector":"#email","valueKey":"email"}}`,
			{ objective: 'Fill form' }
		);

		expect(decision?.action).toEqual({
			type: 'stuck',
			reason: 'Fill/select disallowed: no input values configured'
		});
	});

	it('returns stuck when fill/select selectors fail validation', () => {
		const decision = parseActionDecision(
			`{"action":{"type":"fill","selector":"javascript:alert(1)","valueKey":"email"}}`,
			{ objective: 'Fill form', inputValues: { email: 'me@example.com' } }
		);

		expect(decision?.action).toEqual({
			type: 'stuck',
			reason: 'Fill/select requires selector'
		});
	});

	it('returns undefined for invalid actions', () => {
		expect(parseActionDecision('not json', { objective: 'x' })).toBeUndefined();
		expect(parseActionDecision(`{"action":{"type":"nope"}}`, { objective: 'x' })).toBeUndefined();
		expect(
			parseActionDecision(`{"action":{"type":"wait","ms":-1}}`, {
				objective: 'x'
			})
		).toBeUndefined();
	});

	it('parses scroll actions with optional fields', () => {
		const decision = parseActionDecision(
			`{"action":{"type":"scroll","direction":"down","pixels":200,"selector":"#list"}}`,
			{ objective: 'Scroll' }
		);

		expect(decision?.action).toEqual({
			type: 'scroll',
			direction: 'down',
			pixels: 200,
			selector: '#list'
		});
	});

	it('returns stuck when scroll selectors fail validation', () => {
		const decision = parseActionDecision(
			`{"action":{"type":"scroll","direction":"down","selector":"xpath=//main"}}`,
			{ objective: 'Scroll' }
		);

		expect(decision?.action).toEqual({
			type: 'stuck',
			reason: 'Scroll selector failed validation'
		});
	});
});
