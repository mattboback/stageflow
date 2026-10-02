import { mkdtemp, open, readFile, rm, symlink, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { enforceOutputBudget, MAX_SCANNER_OUTPUT_BYTES } from '../../src/core/output-budget';
describe('scanner artifact disk budget', () => {
	it('removes excessive artifacts and fails the scanner before a clean outcome', async () => {
		const dir = await mkdtemp(join(tmpdir(), 'output-budget-'));
		try {
			const file = await open(join(dir, 'huge.png'), 'w');
			await file.truncate(MAX_SCANNER_OUTPUT_BYTES + 1);
			await file.close();
			await expect(enforceOutputBudget(dir)).rejects.toThrow('output limit');
			await expect(readFile(join(dir, 'huge.png'))).rejects.toThrow();
		} finally {
			await rm(dir, { recursive: true, force: true });
		}
	});
	it('rejects symlinks while retaining bounded normal artifacts', async () => {
		const dir = await mkdtemp(join(tmpdir(), 'output-budget-'));
		try {
			await writeFile(join(dir, 'results.json'), '{}');
			await symlink('/etc/passwd', join(dir, 'secret'));
			await expect(enforceOutputBudget(dir)).rejects.toThrow('output limit');
			expect(await readFile(join(dir, 'results.json'), 'utf8')).toBe('{}');
		} finally {
			await rm(dir, { recursive: true, force: true });
		}
	});
});
