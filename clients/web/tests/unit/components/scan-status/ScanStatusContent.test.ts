import ScanStatusContent from "$lib/components/scan-status/ScanStatusContent.svelte";
import { render } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";

describe("ScanStatusContent", () => {
	it("renders processing view for non-terminal states", () => {
		const { getByText } = render(ScanStatusContent, {
			props: {
				status: "loading",
				result: null,
				logs: [],
			},
		});

		expect(getByText("Processing")).toBeInTheDocument();
		expect(getByText("Initializing scan environment...")).toBeInTheDocument();
	});

	it("renders scanner activity and the Lighthouse long-pole hint", () => {
		const { getByText } = render(ScanStatusContent, {
			props: {
				status: "scanning",
				logs: [],
				result: {
					id: "job-123",
					state: "SCANNING",
					progress: {
						current_page: 1,
						total_pages: 3,
						percentage: 33,
					},
					expected_scanners: ["axe", "lighthouse"],
					completed_scanners: ["axe"],
					remaining_scanners: ["lighthouse"],
					created_at: new Date().toISOString(),
					updated_at: new Date().toISOString(),
				},
			},
		});

		expect(getByText("Scanner Activity")).toBeInTheDocument();
		expect(getByText("axe-core")).toBeInTheDocument();
		expect(getByText("Waiting on Lighthouse")).toBeInTheDocument();
	});
});
