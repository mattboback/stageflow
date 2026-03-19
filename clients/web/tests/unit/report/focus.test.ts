import { getNextFocusIndex } from "$lib/utils";
import { describe, expect, it } from "vitest";

describe("focus trap helper", () => {
	it("wraps forward at end", () => {
		expect(getNextFocusIndex(3, 2, false)).toBe(0);
	});

	it("wraps backward at start", () => {
		expect(getNextFocusIndex(3, 0, true)).toBe(2);
	});

	it("handles invalid active index", () => {
		expect(getNextFocusIndex(3, -1, false)).toBe(0);
		expect(getNextFocusIndex(3, -1, true)).toBe(2);
	});
});
