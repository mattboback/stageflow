const defaultDevApiBase = 'http://localhost:8080';
const defaultProdApiBase = 'http://localhost:3000';

function trimTrailingSlash(value: string): string {
	return value.endsWith('/') ? value.slice(0, -1) : value;
}

function getBrowserOrigin(): string | undefined {
	return typeof globalThis.location?.origin === 'string' && globalThis.location.origin.length > 0
		? globalThis.location.origin
		: undefined;
}

export function resolveApiBase(
	rawApiBase: string | undefined,
	isDev = import.meta.env.DEV,
	origin = getBrowserOrigin() ?? defaultProdApiBase
): string {
	const configuredApiBase = rawApiBase?.trim();
	if (configuredApiBase) {
		return trimTrailingSlash(configuredApiBase);
	}

	return trimTrailingSlash(isDev ? defaultDevApiBase : origin);
}

export const buildApiUrl = (path: string) =>
	`${resolveApiBase(import.meta.env.VITE_API_URL as string | undefined)}${path}`;
