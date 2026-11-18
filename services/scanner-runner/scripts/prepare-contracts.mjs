import { promises as fs } from 'node:fs';
import path from 'node:path';

const contracts = path.resolve(import.meta.dirname, '../../../libs/contracts');
const packages = path.resolve(import.meta.dirname, '../node_modules/@stageflow');

// Bun can install a local contract as a symlink. Replace the link before
// writing the runtime package so the generated files stay untouched.
async function preparePackage(name, files, contents, exports) {
	const source = path.join(contracts, name);
	const target = path.join(packages, 'contracts-' + name);
	for (const filename of Object.keys(files)) {
		await fs.access(path.join(source, filename));
	}
	await fs.rm(target, { recursive: true, force: true });
	await fs.mkdir(target, { recursive: true });
	const manifest = {
		name: '@stageflow/contracts-' + name,
		version: '0.0.0-dev',
		private: true,
		main: './noop.js',
		types: './index.ts',
		exports
	};
	for (const [filename, destination] of Object.entries(files)) {
		const output = path.join(target, destination);
		await fs.mkdir(path.dirname(output), { recursive: true });
		await fs.copyFile(path.join(source, filename), output);
	}
	for (const [filename, content] of Object.entries({
		'package.json': JSON.stringify(manifest, null, 2) + '\n',
		'noop.js': '"use strict";\n',
		...contents
	})) {
		await fs.writeFile(path.join(target, filename), content);
	}
}

await preparePackage(
	'report',
	{
		'generated/typescript/unified-report.v2.ts': 'unified-report.v2.ts'
	},
	{
		'index.ts':
			"export * from './unified-report.v2';\nexport type { UnifiedReportV2 as UnifiedReport } from './unified-report.v2';\n"
	}
);

await preparePackage(
	'scanner-manifest',
	{
		'generated/typescript/index.ts': 'index.ts',
		'generated/typescript/scanner-manifest.ts': 'scanner-manifest.ts',
		'schema/scanner-manifest.schema.json': 'schema/scanner-manifest.schema.json'
	},
	{
		'schema/index.d.ts': 'declare const schema: Record<string, unknown>;\nexport default schema;\n'
	},
	{
		'.': { types: './index.ts', import: './noop.js', require: './noop.js' },
		'./schema': {
			types: './schema/index.d.ts',
			import: './schema/scanner-manifest.schema.json',
			require: './schema/scanner-manifest.schema.json'
		}
	}
);
