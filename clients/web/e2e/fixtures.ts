import fs from 'node:fs';
import path from 'node:path';

import { expect, test as base } from '@playwright/test';

import type { UnifiedReport } from '../app/lib/types/unified-report';

export { expect };

/**
 * The committed multi-scanner contract fixture, the single source of truth these
 * specs mock the API with.
 *
 * Returns a fresh deep copy per call: specs mutate the report to construct
 * scenarios (adding manual-review findings, emptying issues for the all-clear
 * state), and a shared object would leak those edits between files.
 *
 * Typed rather than left as the `any` that JSON.parse yields, so the specs get
 * the same type-checking as the app and the type-aware lint rules can see them.
 */
export function loadReportFixture(): UnifiedReport {
	const fixturePath = path.resolve(
		process.cwd(),
		'../../libs/contracts/report/fixtures/unified-report.v2.all-scans.json'
	);
	return JSON.parse(fs.readFileSync(fixturePath, 'utf8')) as UnifiedReport;
}

export const test = base.extend<{ browserErrors: void }>({
	browserErrors: [
		async ({ context, page }, use) => {
			const errors: string[] = [];
			const monitor = (candidate: typeof page) => {
				candidate.on('pageerror', (error) => errors.push(`pageerror: ${String(error)}`));
				candidate.on('console', (message) => {
					if (message.type() === 'error') errors.push(`console: ${message.text()}`);
				});
			};
			monitor(page);
			context.on('page', monitor);
			await use();
			expect(errors, 'Unexpected browser errors').toEqual([]);
		},
		{ auto: true }
	]
});
