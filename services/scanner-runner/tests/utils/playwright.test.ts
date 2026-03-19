import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { afterEach, describe, expect, it } from "vitest";

import { resolvePlaywrightImageChromiumExecutablePath } from "../../src/utils/playwright";

describe("resolvePlaywrightImageChromiumExecutablePath", () => {
	const originalEnv = process.env;
	const tempRoots: string[] = [];

	afterEach(() => {
		process.env = { ...originalEnv };
		for (const dir of tempRoots.splice(0, tempRoots.length)) {
			fs.rmSync(dir, { recursive: true, force: true });
		}
	});

	it("returns null when no roots exist", () => {
		process.env = { ...originalEnv, PLAYWRIGHT_BROWSERS_PATH: "/no/such/dir" };
		expect(resolvePlaywrightImageChromiumExecutablePath()).toBeNull();
	});

	it("prefers PLAYWRIGHT_BROWSERS_PATH when set", () => {
		const root = fs.mkdtempSync(path.join(os.tmpdir(), "stageflow-pw-"));
		tempRoots.push(root);

		fs.mkdirSync(path.join(root, "chromium-1200", "chrome-linux"), {
			recursive: true,
		});
		fs.writeFileSync(
			path.join(root, "chromium-1200", "chrome-linux", "chrome"),
			"",
		);

		process.env = { ...originalEnv, PLAYWRIGHT_BROWSERS_PATH: root };
		const resolved = resolvePlaywrightImageChromiumExecutablePath();
		expect(resolved).toBe(
			path.join(root, "chromium-1200", "chrome-linux", "chrome"),
		);
	});

	it("picks the highest chromium revision directory", () => {
		const root = fs.mkdtempSync(path.join(os.tmpdir(), "stageflow-pw-"));
		tempRoots.push(root);

		for (const rev of ["chromium-1200", "chromium-1208"]) {
			fs.mkdirSync(path.join(root, rev, "chrome-linux"), { recursive: true });
			fs.writeFileSync(path.join(root, rev, "chrome-linux", "chrome"), "");
		}

		process.env = { ...originalEnv, PLAYWRIGHT_BROWSERS_PATH: root };
		const resolved = resolvePlaywrightImageChromiumExecutablePath();
		expect(resolved).toBe(
			path.join(root, "chromium-1208", "chrome-linux", "chrome"),
		);
	});

	it("falls back to chrome-linux64 when chrome-linux is missing", () => {
		const root = fs.mkdtempSync(path.join(os.tmpdir(), "stageflow-pw-"));
		tempRoots.push(root);

		fs.mkdirSync(path.join(root, "chromium-1300", "chrome-linux64"), {
			recursive: true,
		});
		fs.writeFileSync(
			path.join(root, "chromium-1300", "chrome-linux64", "chrome"),
			"",
		);

		process.env = { ...originalEnv, PLAYWRIGHT_BROWSERS_PATH: root };
		const resolved = resolvePlaywrightImageChromiumExecutablePath();
		expect(resolved).toBe(
			path.join(root, "chromium-1300", "chrome-linux64", "chrome"),
		);
	});
});
