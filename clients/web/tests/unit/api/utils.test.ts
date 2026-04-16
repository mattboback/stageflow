import { describe, expect, it } from 'vitest';

import { resolveApiBase } from '$lib/api/utils';

describe('api/utils resolveApiBase', () => {
	it('preserves an explicit API base URL and trims the trailing slash', () => {
		expect(resolveApiBase('http://localhost:8080/', true, 'http://localhost:3000')).toBe(
			'http://localhost:8080'
		);
	});

	it('defaults to the local API in dev mode when VITE_API_URL is unset', () => {
		expect(resolveApiBase(undefined, true, 'http://localhost:3000')).toBe('http://localhost:8080');
	});

	it('treats blank VITE_API_URL values as unset', () => {
		expect(resolveApiBase('   ', true, 'http://localhost:3000')).toBe('http://localhost:8080');
	});

	it('falls back to the browser origin outside dev mode when VITE_API_URL is unset', () => {
		expect(resolveApiBase(undefined, false, 'https://stageflow.org')).toBe('https://stageflow.org');
	});

	it('throws outside dev mode when no configured base and no origin are available', () => {
		const originalLocation = globalThis.location;
		// @ts-expect-error - deliberately clearing location to simulate SSR
		delete globalThis.location;
		try {
			expect(() => resolveApiBase(undefined, false)).toThrow(/VITE_API_URL is required/);
		} finally {
			globalThis.location = originalLocation;
		}
	});
});
