/**
 * Violation Capture Tests
 *
 * Tests for the core violation screenshot capture logic.
 */

import type { Locator, Page } from "playwright";

import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
	AxeScreenshotConfig,
	ViolationScreenshotCaptureResult,
} from "../../../src/screenshots/axe/types";

import { captureViolationScreenshot } from "../../../src/screenshots/axe/violation-capture";

// Mock node:fs
vi.mock("node:fs", () => ({
	existsSync: vi.fn().mockReturnValue(true),
	mkdirSync: vi.fn(),
}));

// Mock uuid
vi.mock("uuid", () => ({
	v4: vi.fn().mockReturnValue("test-uuid-abc123"),
}));

// Mock config/rule-behaviors
vi.mock("../../../src/config/rule-behaviors", () => ({
	getScreenshotPolicy: vi.fn().mockReturnValue("default"),
}));

// Mock clip module
vi.mock("../../../src/screenshots/axe/clip", () => ({
	computeUnionClip: vi.fn().mockReturnValue({
		clip: { x: 50, y: 50, width: 200, height: 100 },
		elementBounds: [{ x: 10, y: 10, width: 50, height: 30, selector: "#test" }],
	}),
	computeSingleTargetClip: vi.fn().mockReturnValue({
		x: 80,
		y: 80,
		width: 150,
		height: 100,
	}),
	computeElementClip: vi.fn().mockReturnValue({
		x: 100,
		y: 100,
		width: 100,
		height: 50,
	}),
}));

// Mock friendly-node
vi.mock("../../../src/screenshots/axe/friendly-node", () => ({
	buildFriendlyNodeInfo: vi.fn().mockResolvedValue({
		label: "Button 'Submit'",
		tagName: "button",
		role: "button",
		name: "Submit",
		region: "Main content",
	}),
}));

// Mock highlight-css
vi.mock("../../../src/screenshots/axe/highlight-css", () => ({
	injectHighlightCSS: vi.fn().mockResolvedValue(undefined),
	removeHighlightCSS: vi.fn().mockResolvedValue(undefined),
}));

// Mock image module
vi.mock("../../../src/screenshots/axe/image", () => ({
	saveScreenshot: vi.fn().mockResolvedValue(undefined),
	generateThumbnail: vi.fn().mockResolvedValue("thumb-test-uuid.png"),
	compositeOverlay: vi.fn().mockResolvedValue(Buffer.from("overlaid-png")),
}));

// Mock location
vi.mock("../../../src/screenshots/axe/location", () => ({
	captureLocationInfo: vi.fn().mockResolvedValue({
		scrollY: 100,
		viewportHeight: 720,
		docHeight: 2000,
		position: 0.23,
	}),
}));

// Mock semantic-overlay
vi.mock("../../../src/screenshots/axe/semantic-overlay", () => ({
	captureSemanticOverlayScreenshot: vi.fn().mockResolvedValue({
		screenshot: "semantic-test.png",
		thumbnail: "semantic-test-thumb.png",
	}),
}));

// Mock targets
vi.mock("../../../src/screenshots/axe/targets", () => ({
	extractAxeViolationTargets: vi.fn().mockReturnValue({
		targets: ["button.submit", "#form-submit"],
		selector: "button.submit",
	}),
}));

const DEFAULT_CONFIG: AxeScreenshotConfig = {
	screenshotsEnabled: true,
	injectHighlight: true,
	shotScale: 1,
	shotMinWidth: 200,
	shotMinHeight: 150,
	deviceScaleFactor: 1,
	highlightStyle: "dashed",
	mergeTargets: true,
	unionMaxTargets: 10,
	clipPadding: 20,
	contextRatio: 0.5,
	elementContextMultiplier: 2.5,
	elementContextMinSize: 100,
	scrollTimeout: 2000,
	thumbnailSize: 200,
	generateThumbnails: true,
	outputFormat: "png",
	webpQuality: 80,
	overlayMethod: "css-injection",
	semanticOverlayEnabled: true,
	semanticOverlayMaxHeadings: 20,
	semanticOverlayMaxLabelLength: 40,
	semanticOverlayLegendEnabled: true,
};

// Default config for tests
const createConfig = (
	overrides: Partial<AxeScreenshotConfig> = {},
): AxeScreenshotConfig => ({
	...DEFAULT_CONFIG,
	...overrides,
	elementContextMultiplier:
		overrides.elementContextMultiplier ??
		DEFAULT_CONFIG.elementContextMultiplier,
	elementContextMinSize:
		overrides.elementContextMinSize ?? DEFAULT_CONFIG.elementContextMinSize,
});

// Helper to create mock locator
const createMockLocator = (
	boundingBox = { x: 100, y: 100, width: 50, height: 30 },
): Locator => {
	const locator = {
		first: vi.fn().mockReturnThis(),
		scrollIntoViewIfNeeded: vi.fn().mockResolvedValue(undefined),
		boundingBox: vi.fn().mockResolvedValue(boundingBox),
		evaluate: vi.fn().mockResolvedValue({}),
	} as unknown as Locator;
	return locator;
};

// Helper to create mock page
const createMockPage = (overrides: Partial<Page> = {}): Page => {
	return {
		viewportSize: vi.fn().mockReturnValue({ width: 1280, height: 720 }),
		screenshot: vi.fn().mockResolvedValue(Buffer.from("fake-png-data")),
		evaluate: vi.fn().mockResolvedValue({}),
		locator: vi.fn().mockReturnValue(createMockLocator()),
		waitForTimeout: vi.fn().mockResolvedValue(undefined),
		...overrides,
	} as unknown as Page;
};

function expectCaptured(
	result: ViolationScreenshotCaptureResult,
): ViolationScreenshotCaptureResult & { status: "captured" } {
	expect(result.status).toBe("captured");
	if (result.status !== "captured") {
		throw new Error(`expected captured result, got ${result.status}`);
	}

	return result;
}

describe("captureViolationScreenshot", () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	describe("config checks", () => {
		it("returns skipped when screenshotsEnabled is false", async () => {
			const mockPage = createMockPage();
			const config = createConfig({ screenshotsEnabled: false });

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "button-name" },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			expect(result).toEqual({ status: "skipped", reason: "disabled" });
			expect(mockPage.screenshot).not.toHaveBeenCalled();
		});
	});

	describe("policy handling", () => {
		it("delegates to semantic overlay for semantic policy", async () => {
			const { getScreenshotPolicy } = await import(
				"../../../src/config/rule-behaviors"
			);
			const { captureSemanticOverlayScreenshot } = await import(
				"../../../src/screenshots/axe/semantic-overlay"
			);

			(getScreenshotPolicy as ReturnType<typeof vi.fn>).mockReturnValueOnce(
				"semantic",
			);

			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "heading-order" },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			expect(captureSemanticOverlayScreenshot).toHaveBeenCalled();
			const captured = expectCaptured(result);
			expect(captured.screenshot.screenshot).toBe("semantic-test.png");
		});

		it("returns skipped for never policy", async () => {
			const { getScreenshotPolicy } = await import(
				"../../../src/config/rule-behaviors"
			);
			(getScreenshotPolicy as ReturnType<typeof vi.fn>).mockReturnValueOnce(
				"never",
			);

			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "skip-this" },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			expect(result).toEqual({ status: "skipped", reason: "policy_never" });
		});

		it("continues with default capture for default policy", async () => {
			const { getScreenshotPolicy } = await import(
				"../../../src/config/rule-behaviors"
			);
			(getScreenshotPolicy as ReturnType<typeof vi.fn>).mockReturnValueOnce(
				"default",
			);

			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "color-contrast", nodes: [{ target: ["p.low"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			const captured = expectCaptured(result);
			expect(captured.screenshot.screenshot).toBeDefined();
		});
	});

	describe("directory handling", () => {
		it("creates results directory if it does not exist", async () => {
			const { existsSync, mkdirSync } = await import("node:fs");
			(existsSync as ReturnType<typeof vi.fn>).mockReturnValueOnce(false);

			const mockPage = createMockPage();
			const config = createConfig();

			await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#test"] }] },
				resultsDir: "/tmp/new-results",
				cfg: config,
			});

			expect(mkdirSync).toHaveBeenCalledWith("/tmp/new-results", {
				recursive: true,
			});
		});
	});

	describe("screenshot strategies", () => {
		describe("union bounding box strategy", () => {
			it("uses union strategy when mergeTargets is true and multiple targets exist", async () => {
				const { computeUnionClip } = await import(
					"../../../src/screenshots/axe/clip"
				);
				const mockPage = createMockPage();
				const config = createConfig({ mergeTargets: true });

				await captureViolationScreenshot({
					page: mockPage,
					violation: { id: "test", nodes: [{ target: ["#a", "#b"] }] },
					resultsDir: "/tmp/results",
					cfg: config,
				});

				expect(computeUnionClip).toHaveBeenCalled();
			});

			it("falls back when union clip returns null", async () => {
				const { computeUnionClip } = await import(
					"../../../src/screenshots/axe/clip"
				);
				(computeUnionClip as ReturnType<typeof vi.fn>).mockReturnValueOnce(
					null,
				);

				const mockPage = createMockPage();
				const config = createConfig({ mergeTargets: true });

				const result = await captureViolationScreenshot({
					page: mockPage,
					violation: { id: "test", nodes: [{ target: ["#missing"] }] },
					resultsDir: "/tmp/results",
					cfg: config,
				});

				// Should still produce a result via fallback
				expect(result.status).toBe("captured");
				expectCaptured(result);
			});
		});

		describe("single target viewport strategy", () => {
			it("tries single target when selector exists", async () => {
				const { computeUnionClip, computeSingleTargetClip } = await import(
					"../../../src/screenshots/axe/clip"
				);
				(computeUnionClip as ReturnType<typeof vi.fn>).mockReturnValueOnce(
					null,
				);

				const mockPage = createMockPage();
				const config = createConfig({ mergeTargets: true });

				await captureViolationScreenshot({
					page: mockPage,
					violation: { id: "test", nodes: [{ target: ["button.test"] }] },
					resultsDir: "/tmp/results",
					cfg: config,
				});

				expect(computeSingleTargetClip).toHaveBeenCalled();
			});

			it("falls back when single target clip fails", async () => {
				const {
					computeUnionClip,
					computeSingleTargetClip,
					computeElementClip,
				} = await import("../../../src/screenshots/axe/clip");
				(computeUnionClip as ReturnType<typeof vi.fn>).mockReturnValueOnce(
					null,
				);
				(
					computeSingleTargetClip as ReturnType<typeof vi.fn>
				).mockReturnValueOnce(null);

				const mockPage = createMockPage();
				const config = createConfig();

				await captureViolationScreenshot({
					page: mockPage,
					violation: { id: "test", nodes: [{ target: ["#element"] }] },
					resultsDir: "/tmp/results",
					cfg: config,
				});

				expect(computeElementClip).toHaveBeenCalled();
			});
		});

		describe("element screenshot strategy", () => {
			it("uses element clip when viewport strategies fail", async () => {
				const {
					computeUnionClip,
					computeSingleTargetClip,
					computeElementClip,
				} = await import("../../../src/screenshots/axe/clip");
				(computeUnionClip as ReturnType<typeof vi.fn>).mockReturnValueOnce(
					null,
				);
				(
					computeSingleTargetClip as ReturnType<typeof vi.fn>
				).mockReturnValueOnce(null);

				const mockPage = createMockPage();
				const config = createConfig();

				const result = await captureViolationScreenshot({
					page: mockPage,
					violation: { id: "test", nodes: [{ target: ["#fallback"] }] },
					resultsDir: "/tmp/results",
					cfg: config,
				});

				expect(computeElementClip).toHaveBeenCalled();
				expectCaptured(result);
			});

			it("falls back to viewport when element clip fails", async () => {
				const {
					computeUnionClip,
					computeSingleTargetClip,
					computeElementClip,
				} = await import("../../../src/screenshots/axe/clip");
				(computeUnionClip as ReturnType<typeof vi.fn>).mockReturnValueOnce(
					null,
				);
				(
					computeSingleTargetClip as ReturnType<typeof vi.fn>
				).mockReturnValueOnce(null);
				(computeElementClip as ReturnType<typeof vi.fn>).mockReturnValueOnce(
					null,
				);

				const mockPage = createMockPage();
				const config = createConfig();

				const result = await captureViolationScreenshot({
					page: mockPage,
					violation: { id: "test", nodes: [{ target: ["#missing"] }] },
					resultsDir: "/tmp/results",
					cfg: config,
				});

				// Should still succeed with viewport fallback
				expectCaptured(result);
				expect(mockPage.screenshot).toHaveBeenCalledWith({ fullPage: false });
			});
		});

		describe("viewport fallback", () => {
			it("captures viewport screenshot as last resort", async () => {
				const {
					computeUnionClip,
					computeSingleTargetClip,
					computeElementClip,
				} = await import("../../../src/screenshots/axe/clip");
				(computeUnionClip as ReturnType<typeof vi.fn>).mockReturnValueOnce(
					null,
				);
				(
					computeSingleTargetClip as ReturnType<typeof vi.fn>
				).mockReturnValueOnce(null);
				(computeElementClip as ReturnType<typeof vi.fn>).mockReturnValueOnce(
					null,
				);

				// Also make boundingBox return null
				const mockLocator = createMockLocator();
				(mockLocator.boundingBox as ReturnType<typeof vi.fn>).mockResolvedValue(
					null,
				);
				const mockPage = createMockPage({
					locator: vi.fn().mockReturnValue(mockLocator),
				});
				const config = createConfig();

				const result = await captureViolationScreenshot({
					page: mockPage,
					violation: { id: "test", nodes: [{ target: ["#missing"] }] },
					resultsDir: "/tmp/results",
					cfg: config,
				});

				expect(result.status).toBe("captured");
				expectCaptured(result);
			});
		});
	});

	describe("CSS injection", () => {
		it("injects highlight CSS when using css-injection method", async () => {
			const { injectHighlightCSS } = await import(
				"../../../src/screenshots/axe/highlight-css"
			);
			const mockPage = createMockPage();
			const config = createConfig({
				overlayMethod: "css-injection",
				injectHighlight: true,
			});

			await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			expect(injectHighlightCSS).toHaveBeenCalled();
		});

		it("removes highlight CSS in finally block", async () => {
			const { removeHighlightCSS } = await import(
				"../../../src/screenshots/axe/highlight-css"
			);
			const mockPage = createMockPage();
			const config = createConfig({ overlayMethod: "css-injection" });

			await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			expect(removeHighlightCSS).toHaveBeenCalled();
		});

		it("does not inject CSS when using sharp-composite method", async () => {
			const { injectHighlightCSS } = await import(
				"../../../src/screenshots/axe/highlight-css"
			);
			const mockPage = createMockPage();
			const config = createConfig({ overlayMethod: "sharp-composite" });

			await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			expect(injectHighlightCSS).not.toHaveBeenCalled();
		});

		it("applies sharp composite overlay when using sharp-composite method", async () => {
			const { compositeOverlay } = await import(
				"../../../src/screenshots/axe/image"
			);
			const mockPage = createMockPage();
			const config = createConfig({
				overlayMethod: "sharp-composite",
				injectHighlight: true,
			});

			await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			expect(compositeOverlay).toHaveBeenCalled();
		});
	});

	describe("result structure", () => {
		it("returns screenshot filename", async () => {
			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "button-name", nodes: [{ target: ["button"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			const captured = expectCaptured(result);
			expect(captured.screenshot.screenshot).toContain("violation-button-name");
			expect(captured.screenshot.screenshot).toContain(".png");
		});

		it("returns thumbnail filename", async () => {
			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			const captured = expectCaptured(result);
			expect(captured.screenshot.thumbnail).toBeDefined();
		});

		it("returns location info", async () => {
			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			const captured = expectCaptured(result);
			expect(captured.screenshot.locationInfo).toMatchObject({
				scrollY: expect.any(Number),
				viewportHeight: expect.any(Number),
				docHeight: expect.any(Number),
				position: expect.any(Number),
			});
		});

		it("returns friendly node info", async () => {
			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["button.test"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			const captured = expectCaptured(result);
			expect(captured.screenshot.friendlyNode).toBeDefined();
			expect(captured.screenshot.friendlyNode?.label).toBeDefined();
		});

		it("returns element bounds when available", async () => {
			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			const captured = expectCaptured(result);
			expect(captured.screenshot.elementBounds).toBeDefined();
		});
	});

	describe("file output", () => {
		it("uses png extension when outputFormat is png", async () => {
			const mockPage = createMockPage();
			const config = createConfig({ outputFormat: "png" });

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			const captured = expectCaptured(result);
			expect(captured.screenshot.screenshot).toContain(".png");
		});

		it("uses webp extension when outputFormat is webp", async () => {
			const mockPage = createMockPage();
			const config = createConfig({ outputFormat: "webp" });

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			const captured = expectCaptured(result);
			expect(captured.screenshot.screenshot).toContain(".webp");
		});

		it("includes violation ID in filename", async () => {
			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "color-contrast", nodes: [{ target: ["p"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			const captured = expectCaptured(result);
			expect(captured.screenshot.screenshot).toContain("color-contrast");
		});

		it("uses unknown for undefined violation ID", async () => {
			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			const captured = expectCaptured(result);
			expect(captured.screenshot.screenshot).toContain("unknown");
		});
	});

	describe("error handling", () => {
		it("returns failed outcome on general error", async () => {
			const mockPage = {
				viewportSize: vi.fn().mockReturnValue({ width: 1280, height: 720 }),
				screenshot: vi.fn().mockRejectedValue(new Error("Screenshot failed")),
				evaluate: vi.fn().mockResolvedValue({}),
				locator: vi.fn().mockReturnValue({
					first: vi.fn().mockReturnThis(),
					scrollIntoViewIfNeeded: vi
						.fn()
						.mockRejectedValue(new Error("Scroll failed")),
					boundingBox: vi.fn().mockResolvedValue(null),
				}),
				waitForTimeout: vi.fn(),
			} as unknown as Page;
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			expect(result.status).toBe("failed");
		});

		it("cleans up CSS even on error", async () => {
			const { removeHighlightCSS } = await import(
				"../../../src/screenshots/axe/highlight-css"
			);
			const mockPage = {
				viewportSize: vi.fn().mockReturnValue({ width: 1280, height: 720 }),
				screenshot: vi.fn().mockRejectedValue(new Error("Screenshot failed")),
				evaluate: vi.fn().mockResolvedValue({}),
				locator: vi.fn().mockReturnValue({
					first: vi.fn().mockReturnThis(),
					scrollIntoViewIfNeeded: vi.fn().mockResolvedValue(undefined),
					boundingBox: vi.fn().mockResolvedValue(null),
				}),
				waitForTimeout: vi.fn(),
			} as unknown as Page;
			const config = createConfig({ overlayMethod: "css-injection" });

			await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			expect(removeHighlightCSS).toHaveBeenCalled();
		});
	});

	describe("viewport handling", () => {
		it("uses default viewport when viewportSize returns null", async () => {
			const mockPage = createMockPage({
				viewportSize: vi.fn().mockReturnValue(null),
			});
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [{ target: ["#element"] }] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			// Should still succeed with default viewport
			expect(result.status).toBe("captured");
			expectCaptured(result);
		});
	});

	describe("violation with no targets", () => {
		it("handles violation with empty nodes array", async () => {
			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test", nodes: [] },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			// Should still produce viewport fallback
			expect(result.status).toBe("captured");
			expectCaptured(result);
		});

		it("handles violation with undefined nodes", async () => {
			const mockPage = createMockPage();
			const config = createConfig();

			const result = await captureViolationScreenshot({
				page: mockPage,
				violation: { id: "test" },
				resultsDir: "/tmp/results",
				cfg: config,
			});

			expect(result.status).toBe("captured");
			expectCaptured(result);
		});
	});
});
