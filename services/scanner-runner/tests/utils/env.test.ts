import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { getEnvBool, parseEnvBool, parseEnvInt } from '../../src/utils/env';

const ORIGINAL_ENV = process.env;

function resetEnv(): void {
	process.env = { ...ORIGINAL_ENV };
}

describe('env utils', () => {
	beforeEach(() => {
		resetEnv();
	});

	afterEach(() => {
		resetEnv();
	});

	it('parses boolean values with defaults', () => {
		process.env.FLAG = 'true';
		expect(getEnvBool('FLAG', false)).toBe(true);

		process.env.FLAG = '0';
		expect(getEnvBool('FLAG', true)).toBe(false);

		process.env.FLAG = undefined;
		expect(getEnvBool('FLAG', true)).toBe(true);
	});
});

/*
 * The parseEnv* primitives are shared by two policies: the lenient getEnvBool wrapper
 * above, and the fail-fast wrappers in core/config-loader.ts. `undefined` is the
 * agreed signal for "set, but not valid" — the lenient side substitutes a default,
 * the fail-fast side throws. These cases pin that contract for both.
 */
describe('parseEnvBool', () => {
	it('accepts every documented spelling, case- and space-insensitively', () => {
		for (const raw of ['1', 'true', 'yes', 'on', ' TRUE ', 'On']) {
			expect(parseEnvBool(raw)).toBe(true);
		}
		for (const raw of ['0', 'false', 'no', 'off', ' FALSE ', 'Off']) {
			expect(parseEnvBool(raw)).toBe(false);
		}
	});

	it('returns undefined for unset and unrecognized values', () => {
		expect(parseEnvBool(undefined)).toBeUndefined();
		expect(parseEnvBool('')).toBeUndefined();
		expect(parseEnvBool('maybe')).toBeUndefined();
		expect(parseEnvBool('2')).toBeUndefined();
	});
});

describe('parseEnvInt', () => {
	it('accepts non-negative integers only', () => {
		expect(parseEnvInt('0')).toBe(0);
		expect(parseEnvInt('42')).toBe(42);
		expect(parseEnvInt(' 7 ')).toBe(7);
	});

	it('rejects malformed, fractional and negative values', () => {
		// The distinction that matters: config-loader must not read "12abc" as 12.
		expect(parseEnvInt('12abc')).toBeUndefined();
		expect(parseEnvInt('1.5')).toBeUndefined();
		expect(parseEnvInt('-3')).toBeUndefined();
		expect(parseEnvInt(undefined)).toBeUndefined();
	});
});
