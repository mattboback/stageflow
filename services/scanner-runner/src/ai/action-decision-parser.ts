import type { PreScanAction } from '../core/types';
import type { ActionDecision, AgentGoal } from './types';

import { isPlainObject, parseFirstJsonObject } from './json';

const MAX_MODEL_SELECTOR_LENGTH = 512;
const DISALLOWED_SELECTOR_PREFIXES = [
	'//',
	'xpath=',
	'javascript:',
	'data:',
	'file:',
	'http:',
	'https:'
];
export function parseActionDecision(content: string, goal: AgentGoal): ActionDecision | undefined {
	const parsed = parseFirstJsonObject(content);
	if (!parsed) {
		return undefined;
	}

	const rawAction = parsed.action;
	if (!isPlainObject(rawAction)) {
		return undefined;
	}

	const action = parseAction(rawAction, goal);
	if (!action) {
		return undefined;
	}

	const reasoning = typeof parsed.reasoning === 'string' ? parsed.reasoning : '';
	const confidence = typeof parsed.confidence === 'number' ? parsed.confidence : 0.7;

	return { action, reasoning, confidence };
}

function parseAction(
	raw: Record<string, unknown>,
	goal: AgentGoal
): ActionDecision['action'] | undefined {
	const type = raw.type;
	if (typeof type !== 'string') {
		return undefined;
	}

	switch (type) {
		case 'done':
			return { type: 'done' };
		case 'stuck':
			return { type: 'stuck', reason: parseStuckReason(raw) };
		case 'click':
		case 'hover':
			return parseSelectorAction(type, raw);
		case 'wait':
			return parseWaitAction(raw);
		case 'keyboard':
			return parseKeyboardAction(raw);
		case 'scroll':
			return parseScrollAction(raw);
		case 'fill':
		case 'select':
			return parseInputAction(type, raw, goal);
		default:
			return undefined;
	}
}

function parseStuckReason(raw: Record<string, unknown>): string {
	return typeof raw.reason === 'string' ? raw.reason : 'Unknown';
}

function parseSelectorAction(
	type: 'click' | 'hover',
	raw: Record<string, unknown>
): PreScanAction | undefined {
	const selector = parseModelSelector(raw.selector);
	if (!selector) {
		return undefined;
	}

	return type === 'click' ? { type: 'click', selector } : { type: 'hover', selector };
}

function parseWaitAction(raw: Record<string, unknown>): PreScanAction | undefined {
	const ms = raw.ms;
	if (typeof ms !== 'number' || !Number.isFinite(ms) || ms < 0) {
		return undefined;
	}

	return { type: 'wait', ms };
}

function parseKeyboardAction(raw: Record<string, unknown>): PreScanAction | undefined {
	const key = raw.key;
	if (typeof key !== 'string' || !key.trim()) {
		return undefined;
	}

	return { type: 'keyboard', key };
}

function parseScrollAction(raw: Record<string, unknown>): ActionDecision['action'] {
	const direction = raw.direction;
	const pixels = raw.pixels;
	const selector = parseModelSelector(raw.selector);

	const action: PreScanAction = { type: 'scroll' };

	if (typeof direction === 'string' && (direction === 'up' || direction === 'down')) {
		action.direction = direction;
	}

	if (typeof pixels === 'number' && Number.isFinite(pixels) && pixels > 0) {
		action.pixels = pixels;
	}

	if (raw.selector !== undefined && !selector) {
		return { type: 'stuck', reason: 'Scroll selector failed validation' };
	}

	if (selector) {
		action.selector = selector;
	}

	return action;
}

function parseInputAction(
	type: 'fill' | 'select',
	raw: Record<string, unknown>,
	goal: AgentGoal
): ActionDecision['action'] {
	const selector = parseModelSelector(raw.selector);
	if (!selector) {
		return { type: 'stuck', reason: 'Fill/select requires selector' };
	}

	const inputValues = goal.inputValues;
	if (!inputValues || Object.keys(inputValues).length === 0) {
		return {
			type: 'stuck',
			reason: 'Fill/select disallowed: no input values configured'
		};
	}

	const valueKey = raw.valueKey;
	if (typeof valueKey !== 'string' || !valueKey.trim()) {
		return { type: 'stuck', reason: 'Fill/select requires valueKey' };
	}

	const resolved = inputValues[valueKey];
	if (resolved == null) {
		return { type: 'stuck', reason: `Unknown valueKey: ${valueKey}` };
	}

	return type === 'fill'
		? { type: 'fill', selector, value: resolved, valueKey }
		: { type: 'select', selector, value: resolved, valueKey };
}

function parseModelSelector(raw: unknown): string | undefined {
	if (typeof raw !== 'string') {
		return undefined;
	}

	const selector = raw.trim();
	if (!selector || selector.length > MAX_MODEL_SELECTOR_LENGTH) {
		return undefined;
	}

	if (containsControlCharacter(selector)) {
		return undefined;
	}

	const lower = selector.toLowerCase();
	if (DISALLOWED_SELECTOR_PREFIXES.some((prefix) => lower.startsWith(prefix))) {
		return undefined;
	}

	return selector;
}

function containsControlCharacter(value: string): boolean {
	for (let index = 0; index < value.length; index += 1) {
		const code = value.charCodeAt(index);
		if (code <= 0x1f || code === 0x7f) {
			return true;
		}
	}

	return false;
}
