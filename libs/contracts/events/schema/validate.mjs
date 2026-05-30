import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';

const baseDir = path.dirname(fileURLToPath(import.meta.url));
const rootDir = path.resolve(baseDir, '..');
const fixturesDir = path.join(rootDir, 'fixtures');

const schemaByEvent = {
	'scan.completed': 'scan.completed.schema.json',
	'scan.failed': 'scan.failed.schema.json',
	'scan.page.completed': 'scan.page.completed.schema.json'
};

async function readJson(filePath) {
	const raw = await readFile(filePath, 'utf8');
	return JSON.parse(raw);
}

function formatAjvError(error) {
	const location = error.instancePath || '$';
	const message = error.message ?? 'schema validation failed';
	const params = Object.keys(error.params ?? {}).length > 0 ? ` ${JSON.stringify(error.params)}` : '';

	return `${location}: ${message}${params}`;
}

async function main() {
	let failed = false;
	const ajv = new Ajv2020({ strict: false, allErrors: true, verbose: true });
	addFormats(ajv);

	for (const [eventName, schemaFile] of Object.entries(schemaByEvent)) {
		const fixtureFile = `${eventName}.json`;
		const fixturePath = path.join(fixturesDir, fixtureFile);
		const schemaPath = path.join(baseDir, schemaFile);

		const [fixture, schema] = await Promise.all([readJson(fixturePath), readJson(schemaPath)]);
		const validate = ajv.compile(schema);
		if (!validate(fixture)) {
			failed = true;
			console.error(`Validation failed for ${fixtureFile}`);
			for (const error of validate.errors ?? []) {
				console.error(`  - ${formatAjvError(error)}`);
			}
			continue;
		}

		console.log(`Validated ${fixtureFile}`);
	}

	if (failed) {
		process.exit(1);
	}
}

await main();
