/**
 * Environment variable parsing.
 *
 * Two policies exist deliberately, and they are not interchangeable:
 *
 * - The `getEnv*` helpers here are **lenient**: an unrecognized value falls back
 *   to the caller's default. They suit optional presentation tuning, where a typo
 *   should not stop a scan (screenshot format, page-overview scrolling).
 * - `core/config-loader.ts` wraps the same parsers **fail-fast**: an unrecognized
 *   value throws. That is right for job configuration, where silently scanning
 *   with the wrong settings produces a plausible-looking but wrong report.
 *
 * The `parseEnv*` functions are the shared primitives both policies build on, so
 * the accepted spellings of a boolean are defined exactly once. They return
 * `undefined` for "set, but not valid", which each policy then interprets.
 */

const TRUE_TOKENS = ['1', 'true', 'yes', 'on'];
const FALSE_TOKENS = ['0', 'false', 'no', 'off'];

/**
 * Parses a boolean environment value.
 *
 * @returns the boolean, or `undefined` when unset or unrecognized.
 */
export function parseEnvBool(raw: string | undefined): boolean | undefined {
	if (!raw) {
		return undefined;
	}

	const normalized = raw.trim().toLowerCase();
	if (FALSE_TOKENS.includes(normalized)) {
		return false;
	}
	if (TRUE_TOKENS.includes(normalized)) {
		return true;
	}
	return undefined;
}

/**
 * Parses a non-negative integer from an environment value.
 *
 * The whole string must be the integer, so "12abc" and "1.5" are rejected rather
 * than truncated.
 *
 * @returns the integer, or `undefined` when unset or not a non-negative integer.
 */
export function parseEnvInt(raw: string | undefined): number | undefined {
	if (!raw) {
		return undefined;
	}

	const trimmed = raw.trim();
	const parsed = Number.parseInt(trimmed, 10);
	if (!Number.isInteger(parsed) || String(parsed) !== trimmed || parsed < 0) {
		return undefined;
	}
	return parsed;
}

/** Lenient boolean: falls back to `defaultValue` on an unrecognized value. */
export function getEnvBool(name: string, defaultValue: boolean): boolean {
	return parseEnvBool(process.env[name]) ?? defaultValue;
}
