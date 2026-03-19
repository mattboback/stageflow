import { cleanup, render } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it } from "vitest";

import ModalFixture from "../../../fixtures/ModalFixture.svelte";

describe("Modal", () => {
	afterEach(() => {
		cleanup();
	});

	it("focuses initial focus and restores focus on close", async () => {
		const user = userEvent.setup();
		const { getByTestId, queryByRole } = render(ModalFixture);

		const openButton = getByTestId("open-button");
		openButton.focus();

		await user.click(openButton);
		expect(queryByRole("dialog")).toBeInTheDocument();

		const closeButton = getByTestId("close-button");
		expect(document.activeElement).toBe(closeButton);

		await user.keyboard("{Escape}");
		expect(queryByRole("dialog")).not.toBeInTheDocument();
		expect(document.activeElement).toBe(openButton);
	});

	it("closes on backdrop click", async () => {
		const user = userEvent.setup();
		const { getByRole, getByTestId, queryByRole } = render(ModalFixture);

		await user.click(getByTestId("open-button"));
		expect(queryByRole("dialog")).toBeInTheDocument();

		await user.click(getByRole("dialog"));
		expect(queryByRole("dialog")).not.toBeInTheDocument();
	});

	it("does not close when clicking inside content", async () => {
		const user = userEvent.setup();
		const { getByTestId, queryByRole } = render(ModalFixture);

		await user.click(getByTestId("open-button"));
		expect(queryByRole("dialog")).toBeInTheDocument();

		await user.click(getByTestId("modal-body"));
		expect(queryByRole("dialog")).toBeInTheDocument();
	});

	it("traps focus when tabbing", async () => {
		const user = userEvent.setup();
		const { getByTestId } = render(ModalFixture);

		await user.click(getByTestId("open-button"));

		const closeButton = getByTestId("close-button");
		const helpLink = getByTestId("help-link");
		const lastButton = getByTestId("last-button");

		expect(document.activeElement).toBe(closeButton);

		await user.tab();
		expect(document.activeElement).toBe(helpLink);

		await user.tab();
		expect(document.activeElement).toBe(lastButton);

		await user.tab();
		expect(document.activeElement).toBe(closeButton);

		await user.tab({ shift: true });
		expect(document.activeElement).toBe(lastButton);
	});
});
