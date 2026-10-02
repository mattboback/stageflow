import { mkdtemp, readFile, readdir, rm, stat, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import Ajv2020 from 'ajv/dist/2020';
import schema from '../../../../libs/contracts/events/schema/outcome.schema.json';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	MAX_OUTCOME_BYTES,
	MAX_PROGRESS_BYTES,
	ScannerFiles,
	type ScannerOutcome
} from '../../src/core/scanner-files';
import type { PageScanResult, ScanResults } from '../../src/core/types';

const dirs: string[] = [];
async function output(): Promise<string> {
	const dir = await mkdtemp(join(tmpdir(), 'scanner-files-'));
	dirs.push(dir);
	return dir;
}
const severity = { critical: 1, serious: 2, moderate: 0, minor: 0, info: 0 };
function results(failed = 0): ScanResults {
	return {
		jobId: 'test',
		scanner: 'axe',
		version: '1',
		totalPages: 1,
		pages: [{ pageId: 'home' } as PageScanResult],
		summary: {
			totalIssues: 3,
			bySeverity: severity,
			byCategory: {},
			pagesScanned: 1,
			pagesFailed: failed,
			pagesWithIssues: 1,
			avgDurationMs: 1
		},
		startedAt: '',
		completedAt: '',
		durationMs: 1
	};
}
afterEach(async () => {
	await Promise.all(dirs.splice(0).map((dir) => rm(dir, { recursive: true, force: true })));
});
describe('scanner disk outcomes', () => {
	it('atomically replaces an old outcome with validated counts and timing', async () => {
		const dir = await output();
		await writeFile(join(dir, 'outcome.json'), '{}');
		const files = new ScannerFiles(dir);
		await files.publishScanCompleted(results(), {
			totalMs: 50,
			pageIterationMs: 30,
			writeResultsMs: 5,
			publishCompletedMs: 0,
			finalizationMs: 15
		});
		const outcome = JSON.parse(await readFile(join(dir, 'outcome.json'), 'utf8')) as ScannerOutcome;
		expect(new Ajv2020({ strict: false }).compile(schema)(outcome)).toBe(true);
		expect(outcome).toMatchObject({
			status: 'success',
			error: '',
			pages_scanned: 1,
			summary: { total_violations: 3, by_severity: severity },
			timing: { total_ms: 50, page_iteration_ms: 30 }
		});
		expect((await stat(join(dir, 'outcome.json'))).mode & 0o777).toBe(0o600);
		expect(await readdir(dir)).toEqual(['outcome.json']);
	});
	it('marks incomplete page auditing partial', async () => {
		const dir = await output();
		await new ScannerFiles(dir).publishScanCompleted(results(1));
		expect(JSON.parse(await readFile(join(dir, 'outcome.json'), 'utf8'))).toMatchObject({
			status: 'partial',
			error: '1 page audits failed'
		});
	});
	it('keeps audit completion independent of a failed progress write', async () => {
		const dir = await output();
		const logger = { info: vi.fn(), debug: vi.fn(), warn: vi.fn(), error: vi.fn() };
		await writeFile(join(dir, 'progress.jsonl'), '');
		await rm(join(dir, 'progress.jsonl'));
		const invalidProgressDir = join(dir, 'missing');
		const files = new ScannerFiles(invalidProgressDir, logger);
		await files.publishPageCompleted({ pageId: 'home' } as PageScanResult, 1, 1);
		await files.close();
		expect(logger.warn).toHaveBeenCalled();
		await files.publishScanCompleted(results());
		expect(
			JSON.parse(await readFile(join(invalidProgressDir, 'outcome.json'), 'utf8')).status
		).toBe('success');
	});
	it('records failure even with zero pages and bounds error content', async () => {
		const dir = await output();
		const files = new ScannerFiles(dir);
		await files.publishScanFailed('x'.repeat(MAX_OUTCOME_BYTES));
		const data = await readFile(join(dir, 'outcome.json'));
		const outcome = JSON.parse(data.toString()) as ScannerOutcome;
		expect(outcome.status).toBe('failed');
		expect(outcome.pages_scanned).toBe(0);
		expect(outcome.error.length).toBe(4096);
		expect(data.byteLength).toBeLessThan(MAX_OUTCOME_BYTES);
	});
	it('requires successful terminal publication even when progress can be dropped', async () => {
		const dir = await output();
		const file = join(dir, 'file');
		await writeFile(file, '');
		await expect(new ScannerFiles(file).publishScanFailed('failure')).rejects.toThrow();
	});
	it('caps progress across scanner restarts', async () => {
		const dir = await output();
		const path = join(dir, 'progress.jsonl');
		await writeFile(path, 'x'.repeat(MAX_PROGRESS_BYTES));
		const files = new ScannerFiles(dir);
		await files.publishPageCompleted({ pageId: 'home' } as PageScanResult, 1, 1);
		await files.close();
		expect((await stat(path)).size).toBe(MAX_PROGRESS_BYTES);
	});

	it('appends bounded progress without including page contents or credentials', async () => {
		const dir = await output();
		const files = new ScannerFiles(dir);
		await files.publishPageCompleted(
			{
				pageId: 'home',
				url: 'https://secret@example.com',
				rawResults: 'password'
			} as PageScanResult,
			1,
			1
		);
		await files.close();
		const data = await readFile(join(dir, 'progress.jsonl'), 'utf8');
		expect(JSON.parse(data).pages_scanned).toBe(1);
		expect(data).not.toContain('secret');
		expect(data).not.toContain('password');
	});
});
