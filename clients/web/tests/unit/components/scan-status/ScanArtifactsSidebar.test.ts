import ScanArtifactsSidebar from "$lib/components/scan-status/ScanArtifactsSidebar.svelte";
import { render } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";

describe("ScanArtifactsSidebar", () => {
	it("shows locked-open state while scan is in progress", () => {
		const { getByRole, getByText } = render(ScanArtifactsSidebar, {
			props: {
				status: "loading",
				result: null,
			},
		});

		expect(getByText("Artifacts")).toBeInTheDocument();
		expect(getByText("Generating artifacts...")).toBeInTheDocument();

		const toggle = getByRole("button", { name: "In Progress" });
		expect(toggle).toBeDisabled();
	});
});
