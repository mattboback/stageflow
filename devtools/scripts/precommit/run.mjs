#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "..", "..", "..");
const stagedFiles = process.argv
	.slice(2)
	.map((file) => file.replaceAll("\\", "/"));
const monitoredPrefixes = ["libs/contracts/report/"];

const shouldRunContractsReportCheck = stagedFiles.some((file) =>
	monitoredPrefixes.some((prefix) => file.startsWith(prefix)),
);

if (!shouldRunContractsReportCheck) {
	process.exit(0);
}

const bunCommand = process.platform === "win32" ? "bun.exe" : "bun";
const result = spawnSync(bunCommand, ["run", "check"], {
	cwd: resolve(repoRoot, "libs/contracts/report"),
	stdio: "inherit",
});

if (result.error) {
	console.error(result.error.message);
	process.exit(1);
}

process.exit(result.status ?? 1);
