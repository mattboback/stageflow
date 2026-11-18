import { describe, expect, it, vi } from 'vitest';
import { createRedactedLogger } from '../../../src/core/page-iterator-redaction';
import { redactSecretValues } from '../../../src/utils/secret-redaction';
import { runLighthouseInvocation } from '../../../src/scanners/lighthouse/lighthouse-invoker';
import type { LighthouseModule } from '../../../src/scanners/lighthouse/types';

describe('Lighthouse literal log redaction', () => {
	it.each([false, true])('redacts runtime logs, failure=%s', async (fail) => {
		const secret = 'literal p@ss+word';
		const url = `https://example.com/?token=${encodeURIComponent(secret)}`;
		const logs: unknown[] = [];
		const log = (message: string, metadata?: Record<string, unknown>): void => {
			logs.push({ message, metadata });
		};
		const logger = createRedactedLogger({ debug: log, info: log, warn: log, error: log }, (value) =>
			redactSecretValues(value, [secret])
		);
		const lighthouse = vi.fn().mockImplementation(() => {
			if (fail) {
				return Promise.reject(new Error(`failed ${secret} ${url}`));
			}
			return Promise.resolve({
				lhr: { requestedUrl: url, finalUrl: url, categories: {}, audits: {} }
			});
		});
		const run = runLighthouseInvocation({
			url,
			port: 9222,
			categories: [],
			disableStorageReset: true,
			lighthouse: lighthouse as LighthouseModule['default'],
			logger
		});
		const outcome = await run.then(
			() => 'completed',
			() => 'failed'
		);
		expect(outcome).toBe(fail ? 'failed' : 'completed');
		expect(lighthouse).toHaveBeenCalledWith(url, expect.anything(), expect.anything());
		const written = JSON.stringify(logs);
		expect(written).not.toContain(secret);
		expect(written).not.toContain(encodeURIComponent(secret));
		expect(written).toContain('[REDACTED]');
		expect(written).toContain(fail ? 'Lighthouse execution failed' : 'Lighthouse completed');
	});
});
