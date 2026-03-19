#!/usr/bin/env node
/**
 * Validation script for Scanner Manifest JSON Schema
 *
 * Usage:
 *   node validate.js <path-to-manifest.json>
 */

const Ajv = require("ajv");
const addFormats = require("ajv-formats");
const fs = require("node:fs");
const path = require("node:path");

const schemaPath = path.join(__dirname, "scanner-manifest.schema.json");
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
	console.error("Usage: node validate.js <path-to-manifest.json>");
	process.exit(1);
}

if (!fs.existsSync(fileToValidate)) {
	console.error(`Error: File not found: ${fileToValidate}`);
	process.exit(1);
}

const data = JSON.parse(fs.readFileSync(fileToValidate, "utf8"));
const valid = validate(data);

if (!valid) {
	console.error(`❌ ${path.basename(fileToValidate)} is invalid\n`);
	console.error("Validation errors:");
	for (const err of validate.errors) {
		console.error(`  • ${err.instancePath || "root"}: ${err.message}`);
		if (err.params) {
			console.error(`    Params: ${JSON.stringify(err.params, null, 2)}`);
		}
	}
	process.exit(1);
}

const configErrors = validateConfigSchema(data);
const capabilityErrors = validateCapabilities(data);

if (configErrors.length > 0 || capabilityErrors.length > 0) {
	console.error(`❌ ${path.basename(fileToValidate)} has validation errors\n`);
	for (const msg of configErrors) {
		console.error(`  • ${msg}`);
	}
	for (const msg of capabilityErrors) {
		console.error(`  • ${msg}`);
	}
	process.exit(1);
}

console.log(`✅ ${path.basename(fileToValidate)} is valid`);
process.exit(0);

function validateConfigSchema(manifest) {
	if (!manifest || typeof manifest !== "object") {
		return ["Manifest is not an object"];
	}

	if (!Object.prototype.hasOwnProperty.call(manifest, "configSchema")) {
		return [];
	}

	const configSchema = manifest.configSchema;
	if (typeof configSchema === "boolean") {
		return [];
	}

	if (!configSchema || typeof configSchema !== "object") {
		return ["configSchema must be an object or boolean"];
	}

	const compiler = new Ajv({
		strict: false,
		allErrors: true,
		verbose: true,
	});
	addFormats(compiler);

	try {
		compiler.compile(configSchema);
	} catch (err) {
		return [err instanceof Error ? err.message : String(err)];
	}

	return [];
}

function validateCapabilities(manifest) {
	if (!manifest || typeof manifest !== "object") {
		return ["Manifest is not an object"];
	}

	if (!manifest.capabilities || typeof manifest.capabilities !== "object") {
		return [];
	}

	if (
		manifest.capabilities.supportsScreenshots &&
		!manifest.capabilities.requiresBrowser
	) {
		return [
			"capabilities.supportsScreenshots requires capabilities.requiresBrowser to be true",
		];
	}

	return [];
}
