import { afterEach, describe, expect, it, vi } from 'vitest';

const launch = vi.hoisted(() => vi.fn().mockResolvedValue({ port: 9222, pid: 1, kill: vi.fn() }));
vi.mock('chrome-launcher', () => ({ launch }));

import { launchChromeForLighthouse } from '../../../src/scanners/lighthouse/chrome-lifecycle';

afterEach(() => {
	delete process.env.PROVENANCE_AUTH_JSON;
});

describe('launchChromeForLighthouse', () => {
	it('starts Lighthouse Chrome without the scanner environment', async () => {
		process.env.PROVENANCE_AUTH_JSON = '{"secret":"literal"}';
		const logger = { info: vi.fn(), warn: vi.fn(), error: vi.fn(), debug: vi.fn() };

		await launchChromeForLighthouse({ chromePath: '/usr/bin/chromium', logger });

		const options = launch.mock.calls[0]?.[0] as { envVars?: Record<string, string> } | undefined;
		expect(options?.envVars).toBeDefined();
		expect(options?.envVars?.PROVENANCE_AUTH_JSON).toBeUndefined();
		expect(options?.envVars?.DBUS_SESSION_BUS_ADDRESS).toBe('disabled');
	});
});
