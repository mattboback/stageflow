const defaultDevApiBase = 'http://localhost:8080';

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
	origin = getBrowserOrigin()
): string {
	const configuredApiBase = rawApiBase?.trim();
	if (configuredApiBase) {
		return trimTrailingSlash(configuredApiBase);
	}

	if (isDev) {
		return trimTrailingSlash(defaultDevApiBase);
	}

	if (!origin) {
		throw new Error(
			'resolveApiBase: VITE_API_URL is required in production SSR contexts ' +
				'(no browser origin available to fall back to).'
		);
	}

	return trimTrailingSlash(origin);
}

export const buildApiUrl = (path: string) =>
	`${resolveApiBase(import.meta.env.VITE_API_URL as string | undefined)}${path}`;
