#!/usr/bin/env node
/**
 * Validation script for the Provenance JSON Schema.
 *
 * Usage:
 *   node validate.js <path-to-provenance.json>
 */

const Ajv = require("ajv");
const addFormats = require("ajv-formats");
const fs = require("node:fs");
const path = require("node:path");

const schemaPath = path.join(__dirname, "provenance.schema.json");
const schema = JSON.parse(fs.readFileSync(schemaPath, "utf8"));

const ajv = new Ajv({
	strict: false,
	allErrors: true,
	verbose: true,
});
addFormats(ajv);

const validate = ajv.compile(schema);

const fileToValidate = process.argv[2];

if (!fileToValidate) {
	console.error("Usage: node validate.js <path-to-provenance.json>");
	process.exit(1);
}

if (!fs.existsSync(fileToValidate)) {
	console.error(`Error: File not found: ${fileToValidate}`);
	process.exit(1);
}

const data = JSON.parse(fs.readFileSync(fileToValidate, "utf8"));
const valid = validate(data);

if (!valid) {
	console.error(`\u274c ${path.basename(fileToValidate)} is invalid\n`);
	console.error("Validation errors:");
	for (const err of validate.errors) {
		console.error(`  \u2022 ${err.instancePath || "root"}: ${err.message}`);
		if (err.params) {
			console.error(`    Params: ${JSON.stringify(err.params, null, 2)}`);
		}
	}
	process.exit(1);
}

console.log(`\u2705 ${path.basename(fileToValidate)} is valid`);
process.exit(0);
