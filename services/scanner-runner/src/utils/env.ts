/**
 * Environment variable parsing.
 *
 * Two policies exist deliberately, and they are not interchangeable:
 *
 * - The `getEnv*` helpers here are **lenient**: an unrecognized value falls back
 *   to the caller's default. They suit optional presentation tuning, where a typo
 *   should not stop a scan (screenshot geometry, overlay styling).
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
 * Parses a finite number from an environment value.
 *
 * @returns the number, or `undefined` when unset, unparseable, or non-finite.
 */
export function parseEnvNumber(raw: string | undefined): number | undefined {
	if (!raw) {
		return undefined;
	}

	const parsed = Number.parseFloat(raw);
	return Number.isFinite(parsed) ? parsed : undefined;
}

/**
 * Parses a non-negative integer from an environment value.
 *
 * Stricter than `parseEnvNumber`: the whole string must be the integer, so "12abc"
 * and "1.5" are rejected rather than truncated.
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

/** Lenient number: falls back to `defaultValue` on an unparseable value. */
export function getEnvNumber(name: string, defaultValue: number): number {
	return parseEnvNumber(process.env[name]) ?? defaultValue;
}

/**
 * Lenient integer: falls back to `defaultValue` on an unparseable value.
 *
 * Floors a valid non-integer number, where the fail-fast counterpart in
 * config-loader rejects it. Callers here are tuning knobs, so "8.5 columns" is
 * more usefully read as 8 than as a failed scan.
 */
export function getEnvInt(name: string, defaultValue: number): number {
	const parsed = parseEnvNumber(process.env[name]);
	return parsed === undefined ? defaultValue : Math.floor(parsed);
}

/** Trimmed string, or `defaultValue` when unset. */
export function getEnvString(name: string, defaultValue: string): string {
	return process.env[name]?.trim() ?? defaultValue;
}
