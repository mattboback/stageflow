import { mkdtemp, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { afterEach, expect, it, vi } from 'vitest';

const preparation = vi.hoisted(() => vi.fn());
vi.mock('../../../src/screenshots/axe/page-preparation', () => ({
	preparePageForOverview: preparation
}));

afterEach(() => {
	vi.restoreAllMocks();
	vi.resetModules();
});

it('omits raw browser errors when capture fails', async () => {
	const log = vi.spyOn(console, 'log').mockImplementation(() => undefined);
	const { capturePageOverviewRaw } = await import('../../../src/screenshots/axe/page-overview');
	const { loadAxeScreenshotConfig } = await import('../../../src/screenshots/axe/config');
	const canary = 'literal-page-overview-password';
	preparation.mockRejectedValue(
		new Error(`Navigation failed at https://site.test/?token=${canary}`)
	);
	const dir = await mkdtemp(join(tmpdir(), 'stageflow-overview-redaction-'));
	try {
		const result = await capturePageOverviewRaw(
			{},
			[],
			dir,
			'page-1',
			'axe',
			loadAxeScreenshotConfig({ screenshotsEnabled: true }),
			{ enabled: true }
		);
		expect(result).toBeNull();
		expect(JSON.stringify(log.mock.calls)).not.toContain(canary);
	} finally {
		await rm(dir, { recursive: true, force: true });
	}
});
