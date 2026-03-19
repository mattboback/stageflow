import { getVirtualWindow } from "$lib/report";
import { describe, expect, it } from "vitest";

describe("virtualization", () => {
	it("returns zero window for empty input", () => {
		const window = getVirtualWindow({
			scrollTop: 0,
			viewportHeight: 0,
			rowHeight: 40,
			totalItems: 0,
		});
		expect(window.startIndex).toBe(0);
		expect(window.endIndex).toBe(0);
	});

	it("computes window with overscan", () => {
		const window = getVirtualWindow({
			scrollTop: 400,
			viewportHeight: 300,
			rowHeight: 100,
			totalItems: 100,
			overScan: 2,
		});
		expect(window.startIndex).toBe(2);
		expect(window.endIndex).toBeGreaterThan(window.startIndex);
		expect(window.offset).toBe(200);
	});
});
