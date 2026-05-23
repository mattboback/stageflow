#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "..", "..", "..");
const stagedFiles = process.argv
	.slice(2)
	.map((file) => file.replaceAll("\\", "/"));

const contractsToCheck = [
	{ prefix: "libs/contracts/report/", cwd: "libs/contracts/report" },
	{ prefix: "libs/contracts/provenance/", cwd: "libs/contracts/provenance" },
];

const bunCommand = process.platform === "win32" ? "bun.exe" : "bun";

let exitCode = 0;

for (const contract of contractsToCheck) {
	const shouldRun = stagedFiles.some((file) => file.startsWith(contract.prefix));
	if (!shouldRun) {
		continue;
	}

	const result = spawnSync(bunCommand, ["run", "check"], {
		cwd: resolve(repoRoot, contract.cwd),
		stdio: "inherit",
	});

	if (result.error) {
		console.error(result.error.message);
		exitCode = 1;
		continue;
	}

	if ((result.status ?? 1) !== 0) {
		exitCode = result.status ?? 1;
	}
}

process.exit(exitCode);
