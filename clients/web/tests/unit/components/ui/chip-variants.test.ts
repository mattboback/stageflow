import { chipVariants } from "$lib/components/ui/chip";
import { describe, expect, it } from "vitest";

describe("chipVariants", () => {
	it("includes base classes", () => {
		expect(chipVariants()).toContain("rounded-full");
		expect(chipVariants()).toContain("border");
	});

	it("supports active tone", () => {
		expect(chipVariants({ tone: "active" })).toContain("bg-ink");
		expect(chipVariants({ tone: "active" })).toContain("text-surface");
	});

	it("supports success tone", () => {
		expect(chipVariants({ tone: "success" })).toContain("text-emerald-700");
	});

	it("supports ghost tone", () => {
		expect(chipVariants({ tone: "ghost" })).toContain("bg-transparent");
	});

	it("supports caps", () => {
		expect(chipVariants({ caps: true })).toContain("uppercase");
	});

	it("supports xs sizing", () => {
		expect(chipVariants({ size: "xs" })).toContain("px-2");
	});

	it("adds focus styles when interactive", () => {
		expect(chipVariants({ interactive: true })).toContain(
			"focus-visible:ring-2",
		);
	});
});
