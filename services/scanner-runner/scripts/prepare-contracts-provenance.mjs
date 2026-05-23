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
	"provenance",
	"schema",
	"provenance.schema.json",
);

const sourceTypesDir = path.join(
	repoRoot,
	"libs",
	"contracts",
	"provenance",
	"generated",
	"typescript",
);

const sourceIndexPath = path.join(sourceTypesDir, "index.ts");
const sourceProvenanceTypesPath = path.join(sourceTypesDir, "provenance.ts");

const packageRoot = path.join(
	scannerRunnerRoot,
	"node_modules",
	"@stageflow",
	"contracts-provenance",
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
		`Missing provenance schema at ${sourceSchemaPath}. Ensure libs/contracts/provenance is present.`,
	);
}

if (
	!(await pathExists(sourceIndexPath)) ||
	!(await pathExists(sourceProvenanceTypesPath))
) {
	throw new Error(
		`Missing provenance generated types at ${sourceTypesDir}. Run the contracts generator first.`,
	);
}

await fs.rm(packageRoot, { recursive: true, force: true });
await fs.mkdir(path.join(packageRoot, "schema"), { recursive: true });

await fs.writeFile(
	path.join(packageRoot, "package.json"),
	`${JSON.stringify(
		{
			name: "@stageflow/contracts-provenance",
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
					import: "./schema/provenance.schema.json",
					require: "./schema/provenance.schema.json",
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
	path.join(packageRoot, "schema", "provenance.schema.json"),
);

await fs.writeFile(
	path.join(packageRoot, "schema", "index.d.ts"),
	"declare const schema: Record<string, unknown>;\nexport default schema;\n",
);

await fs.cp(sourceIndexPath, path.join(packageRoot, "index.ts"));
await fs.cp(
	sourceProvenanceTypesPath,
	path.join(packageRoot, "provenance.ts"),
);
