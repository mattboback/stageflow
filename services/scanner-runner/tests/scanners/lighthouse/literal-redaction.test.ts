import { describe, expect, it, vi } from 'vitest';
import type { ScanStageLogger } from '../../../src/core/scan-stage-logger';
import { createRedactedLogger } from '../../../src/core/page-iterator-redaction';
import { redactSecretValues } from '../../../src/utils/secret-redaction';
import { runLighthouseInvocation } from '../../../src/scanners/lighthouse/lighthouse-invoker';
import type { LighthouseModule } from '../../../src/scanners/lighthouse/types';

describe('Lighthouse literal log redaction', () => {
	it.each([false, true])('redacts runtime and persisted stage logs, failure=%s', async (fail) => {
		const secret = 'literal p@ss+word';
		const url = `https://example.com/?token=${encodeURIComponent(secret)}`;
		const logs: unknown[] = [];
		const log = (message: string, metadata?: Record<string, unknown>): void => {
			logs.push({ message, metadata });
		};
		const logger = createRedactedLogger({ debug: log, info: log, warn: log, error: log }, (value) =>
			redactSecretValues(value, [secret])
		);
		const recordEvent = vi.fn();
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
			logger,
			redact: (value) => redactSecretValues(value, [secret]),
			scanStageLogger: { recordEvent } as unknown as ScanStageLogger
		});
		const outcome = await run.then(
			() => 'completed',
			() => 'failed'
		);
		expect(outcome).toBe(fail ? 'failed' : 'completed');
		expect(lighthouse).toHaveBeenCalledWith(url, expect.anything(), expect.anything());
		const persisted = JSON.stringify([logs, recordEvent.mock.calls]);
		expect(persisted).not.toContain(secret);
		expect(persisted).not.toContain(encodeURIComponent(secret));
		expect(persisted).toContain('[REDACTED]');
		expect(recordEvent).toHaveBeenCalledWith(
			fail ? 'lighthouse_error' : 'lighthouse_completed',
			expect.anything()
		);
	});
});
