#!/usr/bin/env node

/**
 * Contract drift check for staged changes under libs/contracts/.
 *
 * For each package the commit touches, re-validates its fixtures against its
 * schema and confirms code generation still runs. Both steps matter: a schema edit
 * can invalidate a fixture, and it can break generation outright.
 *
 * This used to shell out to a per-package `scripts/pre-commit-check.sh` that only
 * `report` and `provenance` had — which is exactly why the hook's `files:` pattern
 * excluded `scanner-manifest` and `events`. The steps are identical per package, so
 * they live here once and the list below is the only thing that has to grow when a
 * fifth contract appears.
 *
 * Those scripts also ran their commands under `> /dev/null 2>&1`, so a failure
 * reported "Fixture validation failed" and nothing about the cause. Output is
 * inherited here instead.
 */

import { spawnSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "..", "..", "..");
const stagedFiles = process.argv
	.slice(2)
	.map((file) => file.replaceAll("\\", "/"));

/**
 * `steps` names the package scripts to run, in order. `events` is schema and
 * fixtures only — no TypeScript package and nothing generated — so it validates
 * without generating.
 */
const contractsToCheck = [
	{ dir: "libs/contracts/report", steps: ["validate:fixtures", "generate"] },
	{ dir: "libs/contracts/provenance", steps: ["validate:fixtures", "generate"] },
	{ dir: "libs/contracts/scanner-manifest", steps: ["validate:fixtures", "generate"] },
	{ dir: "libs/contracts/events", steps: ["validate:fixtures"] },
];

const bunCommand = process.platform === "win32" ? "bun.exe" : "bun";

let exitCode = 0;

for (const contract of contractsToCheck) {
	const prefix = `${contract.dir}/`;
	if (!stagedFiles.some((file) => file.startsWith(prefix))) {
		continue;
	}

	const cwd = resolve(repoRoot, contract.dir);

	for (const step of contract.steps) {
		process.stdout.write(`${contract.dir}: ${step}\n`);

		const result = spawnSync(bunCommand, ["run", step], {
			cwd,
			stdio: "inherit",
		});

		if (result.error) {
			console.error(`${contract.dir}: ${step}: ${result.error.message}`);
			exitCode = 1;
			break;
		}

		if ((result.status ?? 1) !== 0) {
			console.error(`${contract.dir}: ${step} failed.`);
			exitCode = result.status ?? 1;
			break;
		}
	}
}

process.exit(exitCode);
