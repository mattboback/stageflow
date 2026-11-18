import { lstat, readdir, rm } from 'node:fs/promises';
import { join } from 'node:path';
import { MAX_PROGRESS_BYTES, MAX_OUTCOME_BYTES } from './scanner-files';

export const MAX_SCANNER_OUTPUT_BYTES = 128 * 1024 * 1024;
/** Bound retained disk artifacts and reject files outside the scanner budget. */
export async function enforceOutputBudget(root: string): Promise<void> {
	// Reserve room for queued best-effort progress and the terminal outcome.
	const artifactLimit = MAX_SCANNER_OUTPUT_BYTES - MAX_PROGRESS_BYTES - MAX_OUTCOME_BYTES;
	let total = 0;
	const exceeded: string[] = [];
	async function visit(dir: string): Promise<void> {
		for (const entry of await readdir(dir, { withFileTypes: true })) {
			const path = join(dir, entry.name);
			if (entry.isDirectory()) {
				await visit(path);
				continue;
			}
			const info = await lstat(path);
			if (!info.isFile() || total + info.size > artifactLimit) {
				await rm(path, { force: true });
				exceeded.push(path);
				continue;
			}
			total += info.size;
		}
	}
	await visit(root);
	if (exceeded.length > 0) {
		throw new Error(`Scanner artifacts exceeded the ${MAX_SCANNER_OUTPUT_BYTES} byte output limit`);
	}
}
