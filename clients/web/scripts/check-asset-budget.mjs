import { readdir, stat } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

const rootDir = fileURLToPath(new URL('..', import.meta.url));
const buildDir = path.join(rootDir, 'build');
const assetPaths = [
	'_app/immutable/assets',
	'_app/immutable/chunks',
	'_app/immutable/entry',
	'_app/immutable/nodes'
];
const budgets = {
	js: 420 * 1024,
	css: 100 * 1024
};

async function sumBytes(extension) {
	let total = 0;

	for (const assetPath of assetPaths) {
		const fullPath = path.join(buildDir, assetPath);
		let entries;

		try {
			entries = await stat(fullPath);
		} catch {
			continue;
		}

		if (!entries.isDirectory()) {
			continue;
		}

		const files = await readdir(fullPath, { withFileTypes: true });

		for (const file of files) {
			if (!file.isFile() || !file.name.endsWith(extension)) {
				continue;
			}

			const filePath = path.join(fullPath, file.name);
			const info = await stat(filePath);
			total += info.size;
		}
	}

	return total;
}

function formatBytes(bytes) {
	return `${(bytes / 1024).toFixed(1)} KiB`;
}

const totals = {
	js: await sumBytes('.js'),
	css: await sumBytes('.css')
};

const failures = Object.entries(totals).filter(([type, total]) => total > budgets[type]);

for (const [type, total] of Object.entries(totals)) {
	console.log(`[asset-budget] ${type}: ${formatBytes(total)} / ${formatBytes(budgets[type])}`);
}

if (failures.length > 0) {
	const message = failures
		.map(([type, total]) => `${type}=${formatBytes(total)} exceeds ${formatBytes(budgets[type])}`)
		.join(', ');

	throw new Error(`StageFlow web asset budget exceeded: ${message}`);
}
