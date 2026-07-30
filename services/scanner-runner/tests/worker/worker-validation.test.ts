import { describe, expect, it } from 'vitest';

import {
	assertScannerIdMatchesManifest,
	assertScannerOptionsMatchSchema
} from '../../src/worker/worker-validation';

describe('worker validation (contract)', () => {
	it('fails in strict mode when manifest.id !== scanner.metadata.name', () => {
		expect(() => {
			assertScannerIdMatchesManifest({
				manifestId: 'axe',
				scannerId: 'axe-v2',
				strict: true
			});
		}).toThrow(/must match/i);
	});

	it('does not fail in non-strict mode when manifest.id !== scanner.metadata.name', () => {
		expect(() => {
			assertScannerIdMatchesManifest({
				manifestId: 'axe',
				scannerId: 'axe-v2',
				strict: false
			});
		}).not.toThrow();
	});

	it('fails in strict mode when SCANNER_OPTIONS violates manifest.configSchema', () => {
		const schema: Record<string, unknown> = {
			type: 'object',
			required: ['goal'],
			properties: {
				goal: {
					type: 'object',
					required: ['objective'],
					properties: { objective: { type: 'string', minLength: 1 } }
				}
			}
		};

		expect(() => {
			assertScannerOptionsMatchSchema({
				manifestId: 'example-scanner',
				schema,
				options: {},
				strict: true
			});
		}).toThrow(/SCANNER_OPTIONS/i);
	});

	it('passes in strict mode when SCANNER_OPTIONS matches manifest.configSchema', () => {
		const schema: Record<string, unknown> = {
			type: 'object',
			required: ['goal'],
			properties: {
				goal: {
					type: 'object',
					required: ['objective'],
					properties: { objective: { type: 'string', minLength: 1 } }
				}
			}
		};

		expect(() => {
			assertScannerOptionsMatchSchema({
				manifestId: 'example-scanner',
				schema,
				options: { goal: { objective: 'Reach checkout' } },
				strict: true
			});
		}).not.toThrow();
	});

	it('passes in strict mode when SCANNER_OPTIONS is missing but schema has no required fields', () => {
		const schema: Record<string, unknown> = {
			type: 'object',
			properties: {
				standard: { type: 'string' }
			}
		};

		expect(() => {
			assertScannerOptionsMatchSchema({
				manifestId: 'axe',
				schema,
				options: undefined,
				strict: true
			});
		}).not.toThrow();
	});

	it('fails with required-field errors (not type errors) when SCANNER_OPTIONS is missing', () => {
		const schema: Record<string, unknown> = {
			type: 'object',
			required: ['goal'],
			properties: {
				goal: {
					type: 'object',
					required: ['objective'],
					properties: { objective: { type: 'string', minLength: 1 } }
				}
			}
		};

		expect(() => {
			assertScannerOptionsMatchSchema({
				manifestId: 'example-scanner',
				schema,
				options: undefined,
				strict: true
			});
		}).toThrow(/required property/i);
	});
});
