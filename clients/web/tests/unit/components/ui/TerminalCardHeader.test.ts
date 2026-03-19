import TerminalCardHeader from "$lib/components/ui/TerminalCardHeader.svelte";
import { cleanup, render } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";

describe("TerminalCardHeader", () => {
	afterEach(() => {
		cleanup();
	});

	it("renders the path", () => {
		const { getByText } = render(TerminalCardHeader, {
			props: { path: "/scan/demo" },
		});
		expect(getByText("/scan/demo")).toBeInTheDocument();
	});

	it("renders badges when provided", () => {
		const { getByText } = render(TerminalCardHeader, {
			props: {
				path: "/scan/demo",
				badges: [
					{ label: "Scan", variant: "terminal" },
					{ label: "Complete", variant: "status" },
				],
			},
		});

		expect(getByText("Scan")).toBeInTheDocument();
		expect(getByText("Complete")).toBeInTheDocument();
	});
});
