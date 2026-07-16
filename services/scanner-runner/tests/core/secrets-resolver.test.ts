import { describe, expect, it } from 'vitest';

import type { Provenance } from '../../src/core/types';

import {
	SecretsResolutionError,
	collectFromEnvReferences,
	createSecretsResolver
} from '../../src/core/secrets-resolver';

describe('SecretsResolver', () => {
	describe('createSecretsResolver', () => {
		it('resolves a literal string value unchanged', () => {
			const resolver = createSecretsResolver({ allowList: [], env: {} });
			expect(resolver.resolveValue('hello')).toBe('hello');
		});

		it('resolves an allow-listed env var', () => {
			const resolver = createSecretsResolver({
				allowList: ['STAGEFLOW_AUTH_USER'],
				env: { STAGEFLOW_AUTH_USER: 'demo@example.com' }
			});
			expect(resolver.resolve({ from_env: 'STAGEFLOW_AUTH_USER' })).toBe('demo@example.com');
			expect(resolver.resolveValue({ from_env: 'STAGEFLOW_AUTH_USER' })).toBe('demo@example.com');
		});

		it('redacts raw, URI-encoded, and form-encoded known values', () => {
			const envSecret = 'p@ss word+1';
			const literalSecret = 'literal value+2';
			const resolver = createSecretsResolver({
				allowList: ['STAGEFLOW_AUTH_PASSWORD'],
				env: { STAGEFLOW_AUTH_PASSWORD: envSecret }
			});
			resolver.resolveValue(literalSecret);

			const formEncodedEnv = new URLSearchParams([['password', envSecret]])
				.toString()
				.slice('password='.length);
			const raw = [
				envSecret,
				encodeURIComponent(envSecret),
				formEncodedEnv,
				encodeURIComponent(literalSecret)
			].join('|');

			expect(resolver.redactKnownValues(raw)).toBe(
				['[REDACTED]', '[REDACTED]', '[REDACTED]', '[REDACTED]'].join('|')
			);
		});

		it('rejects an env var not in the allow-list with SecretsResolutionError', () => {
			const resolver = createSecretsResolver({
				allowList: ['STAGEFLOW_AUTH_USER'],
				env: { OTHER_SECRET: 'leak' }
			});
			expect(() => resolver.resolve({ from_env: 'OTHER_SECRET' })).toThrow(SecretsResolutionError);
		});

		it('rejects an unset env var with SecretsResolutionError', () => {
			const resolver = createSecretsResolver({
				allowList: ['STAGEFLOW_AUTH_PASSWORD'],
				env: {}
			});
			let captured: unknown;
			try {
				resolver.resolve({ from_env: 'STAGEFLOW_AUTH_PASSWORD' });
			} catch (err) {
				captured = err;
			}
			expect(captured).toBeInstanceOf(SecretsResolutionError);
			expect((captured as SecretsResolutionError).reference).toBe('STAGEFLOW_AUTH_PASSWORD');
		});

		it('rejects an empty-string env var (treated as unset)', () => {
			const resolver = createSecretsResolver({
				allowList: ['STAGEFLOW_AUTH_USER'],
				env: { STAGEFLOW_AUTH_USER: '' }
			});
			expect(() => resolver.resolve({ from_env: 'STAGEFLOW_AUTH_USER' })).toThrow(
				SecretsResolutionError
			);
		});

		it('exposes a sorted, frozen allow-list', () => {
			const resolver = createSecretsResolver({
				allowList: ['B', 'A', 'B', 'C'],
				env: {}
			});
			expect(resolver.allowList).toEqual(['A', 'B', 'C']);
			expect(Object.isFrozen(resolver.allowList)).toBe(true);
		});
	});

	describe('collectFromEnvReferences', () => {
		const baseProvenance: Provenance = {
			version: '1.0.0',
			job_id: 'job-1',
			base_url: 'https://app.example.com',
			pages: []
		};

		it('returns an empty array when there are no from_env references', () => {
			expect(collectFromEnvReferences(baseProvenance)).toEqual([]);
		});

		it('collects references from form auth steps', () => {
			const provenance: Provenance = {
				...baseProvenance,
				auth: {
					mode: 'form',
					login_url: 'https://app.example.com/login',
					steps: [
						{
							type: 'fill',
							selector: '#email',
							value: { from_env: 'STAGEFLOW_AUTH_USER' }
						},
						{
							type: 'fill',
							selector: '#password',
							value: { from_env: 'STAGEFLOW_AUTH_PASSWORD' }
						},
						{ type: 'click', selector: 'button[type=submit]' }
					],
					success: { type: 'selector', selector: '[data-test=signed-in]' }
				}
			};

			expect(collectFromEnvReferences(provenance)).toEqual([
				'STAGEFLOW_AUTH_PASSWORD',
				'STAGEFLOW_AUTH_USER'
			]);
		});

		it('collects references from per-page pre_scan_actions and dedupes across pages', () => {
			const provenance: Provenance = {
				...baseProvenance,
				pages: [
					{
						id: 'a',
						path: '/a',
						url: 'https://app.example.com/a',
						pre_scan_actions: [
							{
								type: 'fill',
								selector: '#x',
								value: { from_env: 'TENANT_TOKEN' }
							}
						]
					},
					{
						id: 'b',
						path: '/b',
						url: 'https://app.example.com/b',
						pre_scan_actions: [
							{
								type: 'select',
								selector: '#region',
								value: { from_env: 'TENANT_TOKEN' }
							},
							{
								type: 'fill',
								selector: '#api',
								value: { from_env: 'API_TOKEN' }
							}
						]
					}
				]
			};

			expect(collectFromEnvReferences(provenance)).toEqual(['API_TOKEN', 'TENANT_TOKEN']);
		});

		it('ignores literal-string values', () => {
			const provenance: Provenance = {
				...baseProvenance,
				auth: {
					mode: 'form',
					login_url: 'https://app.example.com/login',
					steps: [{ type: 'fill', selector: '#email', value: 'demo@example.com' }],
					success: { type: 'load' }
				}
			};
			expect(collectFromEnvReferences(provenance)).toEqual([]);
		});

		it('does not produce references from storage_state auth', () => {
			const provenance: Provenance = {
				...baseProvenance,
				auth: { mode: 'storage_state', artifact_key: 'job-1/auth/storage-state.json' }
			};
			expect(collectFromEnvReferences(provenance)).toEqual([]);
		});
	});
});
