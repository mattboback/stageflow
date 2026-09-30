import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { SCANNER_VERSION } from '../../src/scanners/version';

describe('scanner execution version', () => {
	it('records the installed worker version without npm lifecycle variables', () => {
		const manifest = JSON.parse(
			readFileSync(new URL('../../package.json', import.meta.url), 'utf8')
		) as { version: string };
		expect(SCANNER_VERSION).toBe(manifest.version);
		expect(SCANNER_VERSION).toBe('0.2.0');
	});
});
