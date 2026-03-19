/**
 * Fix Examples Utility Tests
 */

import { getFixExample } from "$lib/utils/fix-examples";
import { describe, expect, it } from "vitest";

describe("fix-examples", () => {
	describe("getFixExample", () => {
		describe("known rule IDs", () => {
			it("returns fix example for image-alt", () => {
				const result = getFixExample("image-alt");
				expect(result).not.toBeNull();
				expect(result?.title).toBe("Provide alternative text for images");
				expect(result?.before).toContain("<img");
				expect(result?.after).toContain("alt=");
			});

			it("returns fix example for color-contrast", () => {
				const result = getFixExample("color-contrast");
				expect(result).not.toBeNull();
				expect(result?.title).toBe("Increase text contrast");
				expect(result?.notes).toContain("4.5:1");
			});

			it("returns fix example for html-has-lang", () => {
				const result = getFixExample("html-has-lang");
				expect(result).not.toBeNull();
				expect(result?.title).toBe("Set the page language");
				expect(result?.after).toContain('lang="en"');
			});

			it("returns fix example for language-of-page (alias)", () => {
				const result = getFixExample("language-of-page");
				expect(result).not.toBeNull();
				expect(result?.title).toBe("Set the page language");
			});

			it("returns fix example for html-lang-valid (alias)", () => {
				const result = getFixExample("html-lang-valid");
				expect(result).not.toBeNull();
				expect(result?.title).toBe("Set the page language");
			});

			it("returns fix example for valid-lang (alias)", () => {
				const result = getFixExample("valid-lang");
				expect(result).not.toBeNull();
				expect(result?.title).toBe("Set the page language");
			});

			it("returns fix example for label", () => {
				const result = getFixExample("label");
				expect(result).not.toBeNull();
				expect(result?.title).toBe("Associate labels with form fields");
				expect(result?.before).toContain("<input");
				expect(result?.after).toContain("<label");
			});

			it("returns fix example for link-name", () => {
				const result = getFixExample("link-name");
				expect(result).not.toBeNull();
				expect(result?.title).toBe("Give links an accessible name");
				expect(result?.after).toContain("aria-label=");
			});

			it("returns fix example for button-name", () => {
				const result = getFixExample("button-name");
				expect(result).not.toBeNull();
				expect(result?.title).toBe("Give buttons an accessible name");
				expect(result?.after).toContain("aria-label=");
			});

			it("returns fix example for focus-visible", () => {
				const result = getFixExample("focus-visible");
				expect(result).not.toBeNull();
				expect(result?.title).toBe("Ensure focus is visible");
				expect(result?.after).toContain(":focus-visible");
			});
		});

		describe("unknown rule IDs", () => {
			it("returns null for unknown rule ID", () => {
				expect(getFixExample("unknown-rule")).toBeNull();
			});

			it("returns null for empty string", () => {
				expect(getFixExample("")).toBeNull();
			});

			it("returns null for similar but incorrect rule ID", () => {
				expect(getFixExample("image-alt-text")).toBeNull();
				expect(getFixExample("contrast")).toBeNull();
				expect(getFixExample("lang")).toBeNull();
			});
		});

		describe("FixExample structure", () => {
			it("all known examples have required fields", () => {
				const knownRules = [
					"image-alt",
					"color-contrast",
					"html-has-lang",
					"label",
					"link-name",
					"button-name",
					"focus-visible",
				];

				for (const ruleId of knownRules) {
					const example = getFixExample(ruleId);
					expect(
						example,
						`Expected ${ruleId} to have an example`,
					).not.toBeNull();
					expect(typeof example?.title).toBe("string");
					expect(typeof example?.before).toBe("string");
					expect(typeof example?.after).toBe("string");
					expect(example?.title.length).toBeGreaterThan(0);
					expect(example?.before.length).toBeGreaterThan(0);
					expect(example?.after.length).toBeGreaterThan(0);
				}
			});

			it("notes field is optional but when present is a string", () => {
				const exampleWithNotes = getFixExample("image-alt");
				expect(typeof exampleWithNotes?.notes).toBe("string");
				expect(exampleWithNotes?.notes?.length).toBeGreaterThan(0);
			});

			it("before and after examples are different", () => {
				const knownRules = [
					"image-alt",
					"color-contrast",
					"html-has-lang",
					"label",
					"link-name",
					"button-name",
					"focus-visible",
				];

				for (const ruleId of knownRules) {
					const example = getFixExample(ruleId);
					expect(
						example?.before,
						`Before and after should differ for ${ruleId}`,
					).not.toBe(example?.after);
				}
			});
		});
	});
});
