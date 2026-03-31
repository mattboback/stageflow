import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const baseDir = path.dirname(fileURLToPath(import.meta.url));
const rootDir = path.resolve(baseDir, '..');
const fixturesDir = path.join(rootDir, 'fixtures');

const schemaByEvent = {
	'scan.completed': 'scan.completed.schema.json',
	'scan.failed': 'scan.failed.schema.json',
	'scan.page.completed': 'scan.page.completed.schema.json'
};

function valueType(value) {
	if (Array.isArray(value)) return 'array';
	if (value === null) return 'null';
	return typeof value;
}

function validate(schema, value, pathName = '$') {
	const errors = [];
	const kind = valueType(value);

	if (schema.const !== undefined && value !== schema.const) {
		errors.push(`${pathName}: expected const ${JSON.stringify(schema.const)}, got ${JSON.stringify(value)}`);
	}

	if (schema.type) {
		const schemaType = schema.type;
		if (schemaType === 'integer') {
			if (!(typeof value === 'number' && Number.isInteger(value))) {
				errors.push(`${pathName}: expected integer, got ${kind}`);
				return errors;
			}
		} else if (schemaType === 'object') {
			if (kind !== 'object') {
				errors.push(`${pathName}: expected object, got ${kind}`);
				return errors;
			}
		} else if (schemaType === 'string') {
			if (typeof value !== 'string') {
				errors.push(`${pathName}: expected string, got ${kind}`);
				return errors;
			}
		} else if (schemaType === 'number') {
			if (typeof value !== 'number') {
				errors.push(`${pathName}: expected number, got ${kind}`);
				return errors;
			}
		}
	}

	if (typeof value === 'string' && typeof schema.minLength === 'number' && value.length < schema.minLength) {
		errors.push(`${pathName}: expected minLength ${schema.minLength}, got ${value.length}`);
	}

	if (typeof value === 'number' && typeof schema.minimum === 'number' && value < schema.minimum) {
		errors.push(`${pathName}: expected minimum ${schema.minimum}, got ${value}`);
	}

	if (schema.type === 'object' && schema.required) {
		for (const requiredKey of schema.required) {
			if (!(requiredKey in value)) {
				errors.push(`${pathName}: missing required key '${requiredKey}'`);
			}
		}
	}

	if (schema.type === 'object' && schema.properties) {
		for (const [key, childSchema] of Object.entries(schema.properties)) {
			if (key in value) {
				errors.push(...validate(childSchema, value[key], `${pathName}.${key}`));
			}
		}
	}

	return errors;
}

async function readJson(filePath) {
	const raw = await readFile(filePath, 'utf8');
	return JSON.parse(raw);
}

async function main() {
	let failed = false;

	for (const [eventName, schemaFile] of Object.entries(schemaByEvent)) {
		const fixtureFile = `${eventName}.json`;
		const fixturePath = path.join(fixturesDir, fixtureFile);
		const schemaPath = path.join(baseDir, schemaFile);

		const [fixture, schema] = await Promise.all([readJson(fixturePath), readJson(schemaPath)]);
		const errors = validate(schema, fixture);
		if (errors.length > 0) {
			failed = true;
			console.error(`Validation failed for ${fixtureFile}`);
			for (const error of errors) {
				console.error(`  - ${error}`);
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
