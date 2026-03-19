/**
 * Canonical environment variable parsing utilities.
 * Consolidates duplicate implementations from config-loader.ts and AxeScreenshotService.ts
 */

/**
 * Get a boolean environment variable with a default value.
 * Supports: 0, false, no, off (false) and 1, true, yes, on (true)
 */
export function getEnvBool(name: string, defaultValue: boolean): boolean {
	const raw = process.env[name];
	if (!raw) {
		return defaultValue;
	}

	const normalized = raw.trim().toLowerCase();
	if (["0", "false", "no", "off"].includes(normalized)) {
		return false;
	}
	if (["1", "true", "yes", "on"].includes(normalized)) {
		return true;
	}

	return defaultValue;
}

/**
 * Get a numeric environment variable with a default value.
 */
export function getEnvNumber(name: string, defaultValue: number): number {
	const raw = process.env[name];
	if (!raw) {
		return defaultValue;
	}

	const parsed = Number.parseFloat(raw);
	return Number.isNaN(parsed) ? defaultValue : parsed;
}

/**
 * Get an integer environment variable with a default value.
 */
export function getEnvInt(name: string, defaultValue: number): number {
	return Math.floor(getEnvNumber(name, defaultValue));
}

/**
 * Get a string environment variable with a default value.
 */
export function getEnvString(name: string, defaultValue: string): string {
	return process.env[name]?.trim() ?? defaultValue;
}
