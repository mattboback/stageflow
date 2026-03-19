import { cleanup, render } from "@testing-library/svelte";
import { afterEach, describe, expect, it } from "vitest";

import SelectFieldFixture from "../../../fixtures/SelectFieldFixture.svelte";

describe("SelectField", () => {
	afterEach(() => {
		cleanup();
	});

	it("renders a select with an inline chevron", () => {
		const { getByTestId } = render(SelectFieldFixture);

		const defaultField = getByTestId("select-default");
		expect(defaultField.querySelector("select")).toBeInTheDocument();
		expect(defaultField.querySelector("svg")).toBeInTheDocument();

		const prominentField = getByTestId("select-prominent");
		expect(prominentField.querySelector("select")).toBeInTheDocument();
		expect(prominentField.querySelector("svg")).toBeInTheDocument();
	});

	it("supports prominent variant", () => {
		const { getByTestId } = render(SelectFieldFixture);
		const prominentField = getByTestId("select-prominent");
		const select = prominentField.querySelector("select");

		expect(select).not.toBeNull();
		expect(select as HTMLSelectElement).toHaveClass("border-2");
	});
});
