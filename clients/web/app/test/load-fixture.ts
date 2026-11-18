import fs from 'node:fs';
import path from 'node:path';

import type { UnifiedReport } from '../lib/types/unified-report';

// Vitest runs with cwd at clients/web; the fixture lives in the contracts lib.
const FIXTURE_PATH = path.resolve(
	process.cwd(),
	'../../libs/contracts/report/fixtures/unified-report.v2.all-scans.json'
);

/**
 * The committed all-scans fixture is the canonical example of the v2 report
 * contract; driving component tests with it keeps them honest against the
 * schema instead of hand-rolled partial objects.
 */
export function loadAllScansFixture(): UnifiedReport {
	return JSON.parse(fs.readFileSync(FIXTURE_PATH, 'utf8')) as UnifiedReport;
}
