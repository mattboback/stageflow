import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import {
	getEnvBool,
	getEnvInt,
	getEnvNumber,
	getEnvString,
	parseEnvBool,
	parseEnvInt,
	parseEnvNumber
} from '../../src/utils/env';

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

	it('parses numbers and ints safely', () => {
		process.env.VALUE = '3.14';
		expect(getEnvNumber('VALUE', 1)).toBe(3.14);
		expect(getEnvInt('VALUE', 1)).toBe(3);

		process.env.VALUE = 'nope';
		expect(getEnvNumber('VALUE', 2)).toBe(2);
		expect(getEnvInt('VALUE', 2)).toBe(2);
	});

	it('returns trimmed strings with default fallback', () => {
		process.env.NAME = '  stageflow ';
		expect(getEnvString('NAME', 'default')).toBe('stageflow');

		process.env.NAME = undefined;
		expect(getEnvString('NAME', 'default')).toBe('default');
	});
});

/*
 * The parseEnv* primitives are shared by two policies: the lenient getEnv* wrappers
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

describe('parseEnvNumber', () => {
	it('parses integers, decimals, and negatives', () => {
		expect(parseEnvNumber('42')).toBe(42);
		expect(parseEnvNumber('1.5')).toBe(1.5);
		expect(parseEnvNumber('-3')).toBe(-3);
	});

	it('returns undefined for unset, unparseable, and non-finite values', () => {
		expect(parseEnvNumber(undefined)).toBeUndefined();
		expect(parseEnvNumber('')).toBeUndefined();
		expect(parseEnvNumber('abc')).toBeUndefined();
		expect(parseEnvNumber('Infinity')).toBeUndefined();
	});
});

describe('parseEnvInt', () => {
	it('accepts non-negative integers only', () => {
		expect(parseEnvInt('0')).toBe(0);
		expect(parseEnvInt('42')).toBe(42);
		expect(parseEnvInt(' 7 ')).toBe(7);
	});

	it('rejects what parseEnvNumber would have truncated or allowed', () => {
		// The distinction that matters: config-loader must not read "12abc" as 12.
		expect(parseEnvInt('12abc')).toBeUndefined();
		expect(parseEnvInt('1.5')).toBeUndefined();
		expect(parseEnvInt('-3')).toBeUndefined();
		expect(parseEnvInt(undefined)).toBeUndefined();
	});

	it('differs from the lenient getEnvInt, which floors instead of rejecting', () => {
		process.env.LENIENT = '8.5';
		expect(getEnvInt('LENIENT', 1)).toBe(8);
		expect(parseEnvInt('8.5')).toBeUndefined();
	});
});
