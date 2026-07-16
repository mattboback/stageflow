import { describe, expect, it } from 'vitest';

import type { Provenance } from '../../src/core/types';

import { buildPublicProvenance } from '../../src/core/public-provenance';

const USER_CANARY = 'provenance-user-canary-a17e';
const PASSWORD_CANARY = 'provenance-password-canary-c92b';
const PAGE_LITERAL_CANARY = 'page-literal-canary-f81d';
const PAGE_ENV_CANARY = 'PRIVATE_PAGE_VALUE_CANARY_73B9';

describe('buildPublicProvenance', () => {
	it('omits the complete form recipe while retaining safe auth metadata', () => {
		const executionProvenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-sensitive-form',
			base_url: 'https://example.com',
			pages: [
				{
					id: 'account',
					path: '/account',
					url: 'https://example.com/account',
					pre_scan_actions: [
						{ type: 'fill', selector: '#private-note', value: PAGE_LITERAL_CANARY },
						{
							type: 'select',
							selector: '#private-choice',
							value: { from_env: PAGE_ENV_CANARY }
						}
					]
				}
			],
			auth: {
				mode: 'form',
				login_url: 'https://example.com/login',
				steps: [
					{ type: 'fill', selector: '#username', value: USER_CANARY },
					{ type: 'fill', selector: '#password', value: PASSWORD_CANARY },
					{ type: 'click', selector: 'button[type=submit]' }
				],
				success: { type: 'load' }
			}
		};

		const publicProvenance = buildPublicProvenance(executionProvenance);
		const serialized = JSON.stringify(publicProvenance);

		expect(publicProvenance.auth).toBeUndefined();
		expect(publicProvenance.metadata).toMatchObject({
			auth_configured: true,
			auth_mode: 'form'
		});
		expect(serialized).not.toContain(USER_CANARY);
		expect(serialized).not.toContain(PASSWORD_CANARY);
		expect(serialized).not.toContain(PAGE_LITERAL_CANARY);
		expect(serialized).not.toContain(PAGE_ENV_CANARY);
		expect(serialized).not.toContain('login_url');
		expect(serialized).not.toContain('steps');
		expect(publicProvenance.pages[0]).not.toHaveProperty('pre_scan_actions');
	});

	it('does not expose a storage-state object key', () => {
		const publicProvenance = buildPublicProvenance({
			version: '1.0.0',
			job_id: 'job-storage-state',
			base_url: 'https://example.com',
			pages: [{ id: 'home', path: '/', url: 'https://example.com/' }],
			auth: {
				mode: 'storage_state',
				artifact_key: 'job-storage-state/auth/private-state.json'
			}
		});

		expect(publicProvenance).not.toHaveProperty('auth');
		expect(publicProvenance.metadata).toMatchObject({
			auth_configured: true,
			auth_mode: 'storage_state'
		});
		expect(JSON.stringify(publicProvenance)).not.toContain('private-state.json');
	});
});
