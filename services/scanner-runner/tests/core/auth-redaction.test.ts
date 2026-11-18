/**
 * Redaction test: the literal redactor knows every form and page literal
 * before any of them is used. The files a scanner writes are covered by
 * auth-pipeline-redaction.test.ts.
 */

import { describe, expect, it } from 'vitest';

import type { Provenance } from '../../src/core/types';

import { createLiteralRedactor } from '../../src/core/page-iterator-redaction';

const SECRET_USER = 'auth-test-user-aljkfhqwouihasdf';
const SECRET_PASSWORD = 'auth-test-password-pq3i48hsdfkj2';

function makeProvenance(): Provenance {
	return {
		version: '1.0.0',
		job_id: 'job-redact',
		base_url: 'https://app.example.com',
		pages: [{ id: 'profile', path: '/profile', url: 'https://app.example.com/profile' }],
		auth: {
			mode: 'form',
			login_url: 'https://app.example.com/login',
			steps: [
				{
					type: 'fill',
					selector: 'input[name=email]',
					value: SECRET_USER
				},
				{
					type: 'fill',
					selector: 'input[name=password]',
					value: SECRET_PASSWORD
				},
				{ type: 'click', selector: 'button[type=submit]' }
			],
			success: { type: 'selector', selector: '[data-test=signed-in]' }
		}
	};
}

function expectNoSecrets(haystack: string): void {
	expect(haystack).not.toContain(SECRET_USER);
	expect(haystack).not.toContain(SECRET_PASSWORD);
}

describe('Auth redaction', () => {
	it('collects form and all page literals before execution', () => {
		const provenance = makeProvenance();
		provenance.pages.push({
			id: 'later',
			path: '/',
			url: 'https://app.example.com/',
			pre_scan_actions: [{ type: 'select', selector: '#token', value: 'later literal+value' }]
		});
		const redact = createLiteralRedactor(provenance);
		expectNoSecrets(redact(`${SECRET_USER} ${SECRET_PASSWORD}`));
		expect(redact('later%20literal%2Bvalue')).toBe('[REDACTED]');
	});
});
