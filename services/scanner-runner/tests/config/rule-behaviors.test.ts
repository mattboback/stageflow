import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const ORIGINAL_ENV = process.env;

function resetEnv(): void {
	process.env = { ...ORIGINAL_ENV };
}

describe("rule-behaviors", () => {
	beforeEach(() => {
		resetEnv();
	});

	afterEach(() => {
		resetEnv();
		vi.resetModules();
	});

	it("returns built-in defaults when no rule id is provided", async () => {
		const {
			getRuleBehavior,
			shouldCaptureScreenshot,
			shouldCaptureSemanticScreenshot,
		} = await import("../../src/config/rule-behaviors");
		const behavior = getRuleBehavior();
		expect(behavior.screenshot).toBe("never");
		expect(shouldCaptureScreenshot()).toBe(false);
		expect(shouldCaptureSemanticScreenshot()).toBe(false);
	});

	it("merges overrides from a config file when present", async () => {
		const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "stageflow-rules-"));
		const configPath = path.join(tempDir, "rule-behaviors.json");

		try {
			fs.writeFileSync(
				configPath,
				JSON.stringify({
					defaults: { screenshot: "always" },
					rules: {
						"color-contrast": { owner: "QA", displayMode: "collapsed" },
					},
				}),
				"utf-8",
			);

			process.env.RULE_BEHAVIOR_CONFIG_PATH = configPath;
			vi.resetModules();

			const { getRuleBehavior } = await import(
				"../../src/config/rule-behaviors"
			);
			const behavior = getRuleBehavior("color-contrast");

			expect(behavior.screenshot).toBe("always");
			expect(behavior.owner).toBe("QA");
			expect(behavior.displayMode).toBe("collapsed");
		} finally {
			fs.rmSync(tempDir, { recursive: true, force: true });
		}
	});

	it("falls back to built-ins when override config is invalid", async () => {
		const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "stageflow-rules-"));
		const configPath = path.join(tempDir, "rule-behaviors.json");

		try {
			fs.writeFileSync(configPath, "{invalid", "utf-8");
			process.env.RULE_BEHAVIOR_CONFIG_PATH = configPath;
			vi.resetModules();

			const { getRuleBehavior } = await import(
				"../../src/config/rule-behaviors"
			);
			const behavior = getRuleBehavior("color-contrast");

			expect(behavior.screenshot).toBe("always");
			expect(behavior.owner).toBe("Design");
		} finally {
			fs.rmSync(tempDir, { recursive: true, force: true });
		}
	});

	it("marks heading-order as a semantic screenshot", async () => {
		const {
			getRuleBehavior,
			shouldCaptureScreenshot,
			shouldCaptureSemanticScreenshot,
		} = await import("../../src/config/rule-behaviors");
		const behavior = getRuleBehavior("heading-order");
		expect(behavior.screenshot).toBe("semantic");
		expect(shouldCaptureScreenshot("heading-order")).toBe(false);
		expect(shouldCaptureSemanticScreenshot("heading-order")).toBe(true);
	});

	it("marks label-related rules as semantic screenshots", async () => {
		const {
			getRuleBehavior,
			shouldCaptureScreenshot,
			shouldCaptureSemanticScreenshot,
		} = await import("../../src/config/rule-behaviors");
		const ids = [
			"link-name",
			"button-name",
			"label",
			"form-field-multiple-labels",
		];
		for (const id of ids) {
			const behavior = getRuleBehavior(id);
			expect(behavior.screenshot).toBe("semantic");
			expect(shouldCaptureScreenshot(id)).toBe(false);
			expect(shouldCaptureSemanticScreenshot(id)).toBe(true);
		}
	});
});
