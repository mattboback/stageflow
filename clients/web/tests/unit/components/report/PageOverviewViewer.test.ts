import type { IssueDetail, PageSummary } from "$lib/types/unified-report";

import PageOverviewViewer from "$lib/components/report/PageOverviewViewer.svelte";
import { cleanup, render } from "@testing-library/svelte";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

function createPage(overrides?: Partial<PageSummary>): PageSummary {
	return {
		id: "page-1",
		url: "http://example.com",
		path: "/home",
		issueCount: 1,
		durationMs: 1000,
		pageOverview: {
			screenshotFilename: "shot.png",
			pageWidth: 1200,
			pageHeight: 900,
			elements: [
				{
					issueId: "issue-1",
					ruleId: "color-contrast",
					severity: "critical",
					selector: ".hero",
					nodeIndex: 0,
					xPercent: 10,
					yPercent: 20,
					widthPercent: 30,
					heightPercent: 10,
					x: 120,
					y: 180,
					width: 360,
					height: 90,
				},
			],
		},
		...overrides,
	};
}

function createIssue(overrides?: Partial<IssueDetail>): IssueDetail {
	return {
		id: "issue-1",
		scanner: "axe",
		ruleId: "color-contrast",
		severity: "critical",
		title: "Low contrast",
		description: "Text has low contrast",
		pageId: "page-1",
		pageUrl: "http://example.com",
		elementCount: 1,
		...overrides,
	};
}

describe("PageOverviewViewer", () => {
	afterEach(() => {
		cleanup();
	});

	describe("basic rendering", () => {
		it("renders overlays and triggers selection on click", async () => {
			const user = userEvent.setup();
			const onSelectIssue = vi.fn();

			const { container } = render(PageOverviewViewer, {
				props: {
					page: createPage(),
					issues: [createIssue()],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue,
				},
			});

			const overview = container.querySelector("[data-testid='page-overview']");
			expect(overview).toBeInTheDocument();

			const marker = container.querySelector(
				"[data-testid='page-overview-marker']",
			);
			expect(marker).toBeInTheDocument();
			await user.click(marker as Element);
			expect(onSelectIssue).toHaveBeenCalledWith(
				expect.objectContaining({ id: "issue-1" }),
				"issue-1-el-0",
			);
		});

		it("shows empty state when no screenshot URL is provided", () => {
			const { container, getByText } = render(PageOverviewViewer, {
				props: {
					page: createPage(),
					issues: [createIssue()],
					screenshotUrl: null,
					onSelectIssue: vi.fn(),
				},
			});

			expect(
				getByText(/no page overview screenshot available/i),
			).toBeInTheDocument();
			expect(container.querySelector("svg")).not.toBeInTheDocument();
		});

		it("shows empty state when no elements in pageOverview", () => {
			const { getByText } = render(PageOverviewViewer, {
				props: {
					page: createPage({
						pageOverview: {
							screenshotFilename: "shot.png",
							pageWidth: 1200,
							pageHeight: 900,
							elements: [],
						},
					}),
					issues: [],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			expect(getByText(/no overlay metadata available/i)).toBeInTheDocument();
		});

		it("shows empty state when pageOverview dimensions are invalid", () => {
			const { getByText } = render(PageOverviewViewer, {
				props: {
					page: createPage({
						pageOverview: {
							screenshotFilename: "shot.png",
							pageWidth: 0,
							pageHeight: 0,
							elements: [
								{
									issueId: "issue-1",
									ruleId: "test",
									severity: "critical",
									selector: ".test",
									nodeIndex: 0,
									xPercent: 10,
									yPercent: 10,
									widthPercent: 10,
									heightPercent: 10,
									x: 10,
									y: 10,
									width: 100,
									height: 100,
								},
							],
						},
					}),
					issues: [createIssue()],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			expect(
				getByText(/page overview dimensions not available/i),
			).toBeInTheDocument();
		});
	});

	describe("SVG-based rendering", () => {
		it("renders an SVG element with correct viewBox matching page dimensions", () => {
			const page = createPage();
			const { container } = render(PageOverviewViewer, {
				props: {
					page,
					issues: [createIssue()],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			const svg = container.querySelector("svg");
			expect(svg).toBeInTheDocument();
			expect(svg?.getAttribute("viewBox")).toBe("0 0 1200 900");
		});

		it("renders SVG image element with correct href and dimensions", () => {
			const page = createPage();
			const { container } = render(PageOverviewViewer, {
				props: {
					page,
					issues: [createIssue()],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			const image = container.querySelector("svg image");
			expect(image).toBeInTheDocument();
			expect(image?.getAttribute("href")).toBe("http://example.com/shot.png");
			expect(image?.getAttribute("width")).toBe("1200");
			expect(image?.getAttribute("height")).toBe("900");
		});

		it("renders rect elements at exact pixel coordinates from pageOverview", () => {
			const page = createPage();
			const { container } = render(PageOverviewViewer, {
				props: {
					page,
					issues: [createIssue()],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			const rect = container.querySelector(
				'svg rect[data-testid="page-overview-marker"]',
			);
			expect(rect).toBeInTheDocument();
			// Should use pixel coordinates (x, y, width, height), not percentages
			expect(rect?.getAttribute("x")).toBe("120");
			expect(rect?.getAttribute("y")).toBe("180");
			expect(rect?.getAttribute("width")).toBe("360");
			expect(rect?.getAttribute("height")).toBe("90");
		});

		it("renders multiple overlay rects for multiple issues", () => {
			const page = createPage({
				pageOverview: {
					screenshotFilename: "shot.png",
					pageWidth: 1200,
					pageHeight: 900,
					elements: [
						{
							issueId: "issue-1",
							ruleId: "color-contrast",
							severity: "critical",
							selector: ".hero",
							nodeIndex: 0,
							xPercent: 10,
							yPercent: 20,
							widthPercent: 30,
							heightPercent: 10,
							x: 120,
							y: 180,
							width: 360,
							height: 90,
						},
						{
							issueId: "issue-2",
							ruleId: "image-alt",
							severity: "serious",
							selector: ".logo",
							nodeIndex: 0,
							xPercent: 5,
							yPercent: 5,
							widthPercent: 15,
							heightPercent: 8,
							x: 60,
							y: 45,
							width: 180,
							height: 72,
						},
					],
				},
			});

			const { container } = render(PageOverviewViewer, {
				props: {
					page,
					issues: [
						createIssue({ id: "issue-1", ruleId: "color-contrast" }),
						createIssue({
							id: "issue-2",
							ruleId: "image-alt",
							severity: "serious",
						}),
					],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			const rects = container.querySelectorAll(
				'svg rect[data-testid="page-overview-marker"]',
			);
			expect(rects).toHaveLength(2);
		});
	});

	describe("severity filtering", () => {
		const pageWithMultipleSeverities = createPage({
			pageOverview: {
				screenshotFilename: "shot.png",
				pageWidth: 1200,
				pageHeight: 900,
				elements: [
					{
						issueId: "critical-issue",
						ruleId: "r1",
						severity: "critical",
						selector: ".a",
						nodeIndex: 0,
						xPercent: 10,
						yPercent: 10,
						widthPercent: 10,
						heightPercent: 10,
						x: 100,
						y: 100,
						width: 100,
						height: 100,
					},
					{
						issueId: "moderate-issue",
						ruleId: "r2",
						severity: "moderate",
						selector: ".b",
						nodeIndex: 0,
						xPercent: 20,
						yPercent: 20,
						widthPercent: 10,
						heightPercent: 10,
						x: 200,
						y: 200,
						width: 100,
						height: 100,
					},
				],
			},
		});

		it("shows all severity levels by default", () => {
			const { container } = render(PageOverviewViewer, {
				props: {
					page: pageWithMultipleSeverities,
					issues: [
						createIssue({ id: "critical-issue", severity: "critical" }),
						createIssue({ id: "moderate-issue", severity: "moderate" }),
					],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			const rects = container.querySelectorAll(
				'svg rect[data-testid="page-overview-marker"]',
			);
			expect(rects).toHaveLength(2);
		});

		it("filters out issues when severity chip is toggled off", async () => {
			const user = userEvent.setup();
			const { container, getAllByRole } = render(PageOverviewViewer, {
				props: {
					page: pageWithMultipleSeverities,
					issues: [
						createIssue({ id: "critical-issue", severity: "critical" }),
						createIssue({ id: "moderate-issue", severity: "moderate" }),
					],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			// Toggle off critical severity - find the actual chip button (not the SVG rect)
			const buttons = getAllByRole("button", { name: /^critical$/i });
			const criticalChip = buttons.find(
				(btn) => btn.tagName.toLowerCase() === "button",
			);
			expect(criticalChip).toBeDefined();
			await user.click(criticalChip as Element);

			const rects = container.querySelectorAll(
				'svg rect[data-testid="page-overview-marker"]',
			);
			expect(rects).toHaveLength(1);
		});
	});

	describe("keyboard navigation", () => {
		it("responds to keyboard events on the SVG for accessibility", async () => {
			const user = userEvent.setup();
			const onSelectIssue = vi.fn();

			const page = createPage({
				pageOverview: {
					screenshotFilename: "shot.png",
					pageWidth: 1200,
					pageHeight: 900,
					elements: [
						{
							issueId: "issue-1",
							ruleId: "r1",
							severity: "critical",
							selector: ".a",
							nodeIndex: 0,
							xPercent: 10,
							yPercent: 10,
							widthPercent: 10,
							heightPercent: 10,
							x: 100,
							y: 100,
							width: 100,
							height: 100,
						},
						{
							issueId: "issue-2",
							ruleId: "r2",
							severity: "serious",
							selector: ".b",
							nodeIndex: 0,
							xPercent: 20,
							yPercent: 20,
							widthPercent: 10,
							heightPercent: 10,
							x: 200,
							y: 200,
							width: 100,
							height: 100,
						},
					],
				},
			});

			const { container } = render(PageOverviewViewer, {
				props: {
					page,
					issues: [
						createIssue({ id: "issue-1", severity: "critical" }),
						createIssue({ id: "issue-2", severity: "serious" }),
					],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue,
				},
			});

			const keyboardTarget = container.querySelector(
				"[data-testid='page-overview-keyboard']",
			);
			expect(keyboardTarget).toBeInTheDocument();

			if (!(keyboardTarget instanceof HTMLElement)) {
				throw new Error("expected keyboard target to be an HTMLElement");
			}

			// Focus the container and navigate
			keyboardTarget.focus();
			await user.keyboard("{ArrowRight}");
			await user.keyboard("{Enter}");

			expect(onSelectIssue).toHaveBeenCalled();
		});

		it("triggers selection when Enter is pressed on a rect", () => {
			const onSelectIssue = vi.fn();

			const { container } = render(PageOverviewViewer, {
				props: {
					page: createPage(),
					issues: [createIssue()],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue,
				},
			});

			const rect = container.querySelector(
				'svg rect[data-testid="page-overview-marker"]',
			);
			expect(rect).toBeInTheDocument();

			// Simulate keyboard interaction on the rect
			rect?.dispatchEvent(
				new KeyboardEvent("keydown", {
					key: "Enter",
					bubbles: true,
					cancelable: true,
				}),
			);

			expect(onSelectIssue).toHaveBeenCalledWith(
				expect.objectContaining({ id: "issue-1" }),
				"issue-1-el-0",
			);
		});
	});

	describe("zoom controls", () => {
		it("renders zoom control buttons", () => {
			const { getByTitle } = render(PageOverviewViewer, {
				props: {
					page: createPage(),
					issues: [createIssue()],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			expect(getByTitle("Zoom out")).toBeInTheDocument();
			expect(getByTitle("Zoom in")).toBeInTheDocument();
			expect(getByTitle("Reset zoom")).toBeInTheDocument();
		});

		it("updates zoom level when zoom buttons are clicked", async () => {
			const user = userEvent.setup();
			const { getByTitle, getByText } = render(PageOverviewViewer, {
				props: {
					page: createPage(),
					issues: [createIssue()],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			// Initial zoom is 100%
			expect(getByText("100%")).toBeInTheDocument();

			// Zoom in
			await user.click(getByTitle("Zoom in"));
			expect(getByText("120%")).toBeInTheDocument();

			// Zoom out
			await user.click(getByTitle("Zoom out"));
			expect(getByText("100%")).toBeInTheDocument();

			// Reset zoom
			await user.click(getByTitle("Zoom in"));
			await user.click(getByTitle("Zoom in"));
			await user.click(getByTitle("Reset zoom"));
			expect(getByText("100%")).toBeInTheDocument();
		});
	});

	describe("accessibility", () => {
		it("has appropriate ARIA labels on interactive elements", () => {
			const { container } = render(PageOverviewViewer, {
				props: {
					page: createPage(),
					issues: [createIssue()],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			const svg = container.querySelector("svg");
			expect(svg?.getAttribute("role")).toBe("img");
			expect(svg?.getAttribute("aria-label")).toContain("highlighted issues");

			const rect = container.querySelector(
				'svg rect[data-testid="page-overview-marker"]',
			);
			expect(rect?.getAttribute("role")).toBe("button");
			expect(rect?.getAttribute("aria-label")).toContain("Low contrast");
			expect(rect?.getAttribute("aria-label")).toContain("critical");
		});

		it("renders title elements within rects for tooltips", () => {
			const { container } = render(PageOverviewViewer, {
				props: {
					page: createPage(),
					issues: [createIssue()],
					screenshotUrl: "http://example.com/shot.png",
					onSelectIssue: vi.fn(),
				},
			});

			const title = container.querySelector(
				'svg rect[data-testid="page-overview-marker"] title',
			);
			expect(title).toBeInTheDocument();
			expect(title?.textContent).toContain("Low contrast");
		});
	});
});
