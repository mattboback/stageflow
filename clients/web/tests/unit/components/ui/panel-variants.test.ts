import { panelVariants } from "$lib/components/ui/panel";
import { describe, expect, it } from "vitest";

describe("panelVariants", () => {
	it("includes base panel classes", () => {
		expect(panelVariants({})).toContain("border-line");
		expect(panelVariants({})).toContain("bg-surface");
	});

	it("applies padding/rounded/interactive variants", () => {
		expect(panelVariants({ padding: "xs" })).toContain("p-2");
		expect(panelVariants({ padding: "md" })).toContain("p-5");
		expect(panelVariants({ rounded: "3xl" })).toContain("rounded-3xl");
		expect(panelVariants({ interactive: true })).toContain("hover-glow");
	});

	it("applies surface variants", () => {
		expect(panelVariants({ variant: "muted" })).toContain("bg-surface-muted");
	});
});
