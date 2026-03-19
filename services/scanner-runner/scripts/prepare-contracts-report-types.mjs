import { promises as fs } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const scannerRunnerRoot = path.resolve(__dirname, "..");
const repoRoot = path.resolve(scannerRunnerRoot, "../..");

const sourceTypesDir = path.join(
	repoRoot,
	"libs",
	"contracts",
	"report",
	"generated",
	"typescript",
);

const packageRoot = path.join(
	scannerRunnerRoot,
	"node_modules",
	"@stageflow",
	"contracts-report",
);

async function pathExists(targetPath) {
	try {
		await fs.stat(targetPath);
		return true;
	} catch {
		return false;
	}
}

if (!(await pathExists(sourceTypesDir))) {
	throw new Error(
		`Missing contracts-report generated types at ${sourceTypesDir}. Run the repo generator/build for @stageflow/contracts-report first.`,
	);
}

const unifiedReportPath = path.join(sourceTypesDir, "unified-report.v2.ts");
if (!(await pathExists(unifiedReportPath))) {
	throw new Error(`Missing ${unifiedReportPath}.`);
}

await fs.rm(packageRoot, { recursive: true, force: true });
await fs.mkdir(packageRoot, { recursive: true });

await fs.writeFile(
	path.join(packageRoot, "package.json"),
	`${JSON.stringify(
		{
			name: "@stageflow/contracts-report",
			version: "0.0.0-dev",
			private: true,
			main: "./noop.js",
			types: "./index.ts",
		},
		null,
		2,
	)}\n`,
);

await fs.writeFile(path.join(packageRoot, "noop.js"), '"use strict";\n');

await fs.cp(unifiedReportPath, path.join(packageRoot, "unified-report.v2.ts"));

await fs.writeFile(
	path.join(packageRoot, "index.ts"),
	[
		"export * from './unified-report.v2';",
		"export type { UnifiedReportV2 as UnifiedReport } from './unified-report.v2';",
		"",
	].join("\n"),
);
