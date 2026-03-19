import { describe, expect, it } from "vitest";

import { escapeHtml } from "../../src/utils/html";

describe("html", () => {
	describe("escapeHtml", () => {
		it("returns empty string for undefined or null", () => {
			expect(escapeHtml(undefined)).toBe("");
			expect(escapeHtml(null)).toBe("");
		});

		it("escapes special characters", () => {
			expect(escapeHtml('<script>alert("xss")</script>')).toBe(
				"&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;",
			);
		});

		it("escapes ampersands", () => {
			expect(escapeHtml("Tom & Jerry")).toBe("Tom &amp; Jerry");
		});

		it("escapes single quotes", () => {
			expect(escapeHtml("It's me")).toBe("It&#39;s me");
		});

		it("converts numbers to string", () => {
			expect(escapeHtml(123)).toBe("123");
		});
	});
});
