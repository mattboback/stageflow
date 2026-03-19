import { promises as fs } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const scannerRunnerRoot = path.resolve(__dirname, "..");
const repoRoot = path.resolve(scannerRunnerRoot, "../..");

const sourceSchemaPath = path.join(
	repoRoot,
	"libs",
	"contracts",
	"scanner-manifest",
	"schema",
	"scanner-manifest.schema.json",
);

const sourceTypesDir = path.join(
	repoRoot,
	"libs",
	"contracts",
	"scanner-manifest",
	"generated",
	"typescript",
);

const sourceIndexPath = path.join(sourceTypesDir, "index.ts");
const sourceManifestTypesPath = path.join(
	sourceTypesDir,
	"scanner-manifest.ts",
);

const packageRoot = path.join(
	scannerRunnerRoot,
	"node_modules",
	"@stageflow",
	"contracts-scanner-manifest",
);

async function pathExists(targetPath) {
	try {
		await fs.stat(targetPath);
		return true;
	} catch {
		return false;
	}
}

if (!(await pathExists(sourceSchemaPath))) {
	throw new Error(
		`Missing scanner manifest schema at ${sourceSchemaPath}. Ensure libs/contracts/scanner-manifest is present.`,
	);
}

if (
	!(await pathExists(sourceIndexPath)) ||
	!(await pathExists(sourceManifestTypesPath))
) {
	throw new Error(
		`Missing scanner manifest generated types at ${sourceTypesDir}. Run the contracts generator first.`,
	);
}

await fs.rm(packageRoot, { recursive: true, force: true });
await fs.mkdir(path.join(packageRoot, "schema"), { recursive: true });

await fs.writeFile(
	path.join(packageRoot, "package.json"),
	`${JSON.stringify(
		{
			name: "@stageflow/contracts-scanner-manifest",
			version: "0.0.0-dev",
			private: true,
			main: "./noop.js",
			types: "./index.ts",
			exports: {
				".": {
					types: "./index.ts",
					import: "./noop.js",
					require: "./noop.js",
				},
				"./schema": {
					types: "./schema/index.d.ts",
					import: "./schema/scanner-manifest.schema.json",
					require: "./schema/scanner-manifest.schema.json",
				},
			},
		},
		null,
		2,
	)}\n`,
);

await fs.writeFile(path.join(packageRoot, "noop.js"), '"use strict";\n');

await fs.cp(
	sourceSchemaPath,
	path.join(packageRoot, "schema", "scanner-manifest.schema.json"),
);

await fs.writeFile(
	path.join(packageRoot, "schema", "index.d.ts"),
	"declare const schema: Record<string, unknown>;\nexport default schema;\n",
);

await fs.cp(sourceIndexPath, path.join(packageRoot, "index.ts"));
await fs.cp(
	sourceManifestTypesPath,
	path.join(packageRoot, "scanner-manifest.ts"),
);
