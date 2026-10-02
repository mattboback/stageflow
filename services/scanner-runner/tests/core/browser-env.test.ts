import { describe, expect, it } from 'vitest';

import { browserEnv } from '../../src/core/browser-env';

describe('browserEnv', () => {
	it('passes Chromium only what it needs to run', () => {
		const env = browserEnv({
			PATH: '/usr/bin',
			HOME: '/tmp',
			TZ: 'UTC',
			PROVENANCE_AUTH_JSON: '{"steps":[{"value":"secret"}]}',
			SCANNER_OPTIONS: '{"token":"secret"}',
			DATABASE_URL: 'postgres://user:secret@db/stageflow',
			NODE_OPTIONS: '--require /tmp/evil.js'
		});

		expect(env).toEqual({
			DBUS_SESSION_BUS_ADDRESS: 'disabled',
			PATH: '/usr/bin',
			HOME: '/tmp',
			TZ: 'UTC'
		});
	});
});
