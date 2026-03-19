import {
	modalContentVariants,
	modalOverlayVariants,
} from "$lib/components/ui/modal";
import { describe, expect, it } from "vitest";

describe("modalOverlayVariants", () => {
	it("includes base overlay positioning", () => {
		expect(modalOverlayVariants({})).toContain("fixed");
		expect(modalOverlayVariants({})).toContain("inset-0");
	});

	it("applies backdrop/layout/padding/z variants", () => {
		expect(modalOverlayVariants({ backdrop: "dark" })).toContain("bg-black/80");
		expect(modalOverlayVariants({ layout: "fullscreen" })).toContain("flex");
		expect(modalOverlayVariants({ layout: "fullscreen" })).toContain(
			"flex-col",
		);
		expect(modalOverlayVariants({ padding: "sm" })).toContain("p-2");
		expect(modalOverlayVariants({ z: 40 })).toContain("z-40");
	});
});

describe("modalContentVariants", () => {
	it("applies size variants", () => {
		expect(modalContentVariants({ size: "lg" })).toContain("max-w-lg");
		expect(modalContentVariants({ size: "xl" })).toContain("max-w-3xl");
		expect(modalContentVariants({ size: "auto" })).toContain("w-auto");
	});
});
