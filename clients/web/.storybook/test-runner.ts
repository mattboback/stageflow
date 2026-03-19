import type { TestRunnerConfig } from "@storybook/test-runner";

import { checkA11y, configureAxe, injectAxe } from "axe-playwright";

const A11Y_TARGET_SELECTORS = ["#storybook-root", "#storybook-docs"];

async function resolveA11yTarget(
	page: Parameters<NonNullable<TestRunnerConfig["postVisit"]>>[0],
) {
	await page.waitForFunction(
		(selectors) =>
			selectors.some((selector) => document.querySelector(selector) !== null),
		A11Y_TARGET_SELECTORS,
		{ timeout: 5_000 },
	);

	for (const selector of A11Y_TARGET_SELECTORS) {
		if ((await page.locator(selector).count()) > 0) {
			return selector;
		}
	}

	throw new Error(
		`Unable to locate Storybook render root for accessibility checks at ${page.url()}`,
	);
}

const testRunnerConfig: TestRunnerConfig = {
	async preVisit(page) {
		await injectAxe(page);
	},
	async postVisit(page) {
		const a11yTargetSelector = await resolveA11yTarget(page);

		await configureAxe(page, {
			rules: [
				// Storybook wraps stories in its own landmark containers.
				{ id: "region", enabled: false },
			],
		});

		await checkA11y(page, a11yTargetSelector, {
			detailedReport: true,
			detailedReportOptions: {
				html: true,
			},
			axeOptions: {
				runOnly: {
					type: "tag",
					values: ["wcag2a", "wcag2aa"],
				},
			},
		});
	},
};

export default testRunnerConfig;
