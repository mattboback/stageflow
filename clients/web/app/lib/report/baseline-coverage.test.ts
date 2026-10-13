import { describe, expect, it } from 'vitest';
import { loadAllScansFixture } from '../../test/load-fixture';
import { baselineCoverageProblem } from './baseline-coverage';

function fixture() {
	const report = loadAllScansFixture();
	report.meta.scanContext = {
		expectedPages: report.pages.map((page) => page.id),
		identityScope: 'path',
		scannerConfigFingerprints: {},
		purpose: 'full',
		authenticated: false,
		inputType: 'urls'
	};
	report.scanners.forEach((scanner) => {
		scanner.status = 'success';
		scanner.pages = report.pages.map((page) => ({
			pageId: page.id,
			status: 'success',
			rules: [{ ruleId: 'manual-check', outcome: 'incomplete' }]
		}));
	});
	return report;
}
describe('baseline coverage', () => {
	it('allows a complete scan with manual incomplete rule outcomes', () => {
		expect(baselineCoverageProblem(fixture())).toBeNull();
	});
	it.each(['failed', 'skipped', 'partial'] as const)('rejects %s scanners', (status) => {
		const report = fixture();
		report.scanners[0]!.status = status;
		expect(baselineCoverageProblem(report)).toContain('incomplete scanner coverage');
	});
	it('rejects a missing expected page even if scanner coverage claims success', () => {
		const report = fixture();
		report.pages.pop();
		expect(baselineCoverageProblem(report)).toContain('missing expected pages');
	});
	it('rejects missing or unsuccessful page coverage', () => {
		const report = fixture();
		report.scanners[0]!.pages = [];
		expect(baselineCoverageProblem(report)).toContain('missing or unsuccessful page checks');
	});
	it('keeps older successful reports reviewable as baselines', () => {
		const report = loadAllScansFixture();
		delete report.meta.scanContext;
		expect(baselineCoverageProblem(report)).toBeNull();
	});
});
