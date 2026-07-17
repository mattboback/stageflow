import { describe, expect, it } from 'vitest';

import {
	redactDynamicStringValues,
	redactSecretValues,
	redactStringValues
} from '../../src/utils/secret-redaction';

describe('secret redaction', () => {
	it('preserves structural keys while redacting string values', () => {
		const redact = (value: string): string => redactSecretValues(value, ['a', 's']);
		const result = redactStringValues(
			{
				pageId: 'page-1',
				url: 'https://example.com',
				path: '/page',
				success: false,
				issues: [{ description: 'a secret' }],
				rawResults: { sample: 's' }
			},
			redact
		);

		expect(Object.keys(result)).toEqual([
			'pageId',
			'url',
			'path',
			'success',
			'issues',
			'rawResults'
		]);
		expect(result.issues).toHaveLength(1);
		expect(result.pageId).toContain('[REDACTED]');
		expect(result.rawResults.sample).toBe('[REDACTED]');
	});

	it('redacts encoded dynamic keys without losing colliding values', () => {
		const secret = 'p@ss word+1';
		const uriEncoded = encodeURIComponent(secret);
		const formEncoded = new URLSearchParams([['value', secret]]).toString().slice('value='.length);
		const redact = (value: string): string => redactSecretValues(value, [secret]);

		const result = redactDynamicStringValues(
			{
				[secret]: 'raw',
				[uriEncoded]: 'uri',
				[formEncoded]: 'form'
			},
			redact
		);

		expect(result).toEqual({
			'[REDACTED]': 'raw',
			'[REDACTED]#2': 'uri',
			'[REDACTED]#3': 'form'
		});
		const serialized = JSON.stringify(result);
		expect(serialized).not.toContain(secret);
		expect(serialized).not.toContain(uriEncoded);
		expect(serialized).not.toContain(formEncoded);
	});
});
