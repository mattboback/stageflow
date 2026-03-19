import { describe, expect, it } from "vitest";

import type { PageOverviewElement } from "../../../src/screenshots/axe/types";

import {
  buildPageOverviewOverlaySvg,
  SEVERITY_COLORS,
} from "../../../src/screenshots/axe/page-overview-overlay";

describe("page overview overlay", () => {
  it("renders one rect per element", () => {
    const elements: PageOverviewElement[] = [
      {
        issueId: "i1",
        ruleId: "r1",
        severity: "critical",
        selector: "#a",
        nodeIndex: 0,
        xPercent: 0.1,
        yPercent: 0.2,
        widthPercent: 0.3,
        heightPercent: 0.4,
        x: 10,
        y: 20,
        width: 30,
        height: 40,
      },
      {
        issueId: "i2",
        ruleId: "r2",
        severity: "minor",
        selector: "#b",
        nodeIndex: 0,
        xPercent: 0.1,
        yPercent: 0.2,
        widthPercent: 0.3,
        heightPercent: 0.4,
        x: 1,
        y: 2,
        width: 3,
        height: 4,
      },
    ];

    const svg = buildPageOverviewOverlaySvg(elements, 100, 200).toString("utf-8");
    const rectCount = (svg.match(/<rect\b/g) ?? []).length;
    expect(rectCount).toBe(2);
    expect(svg).toContain('width="100"');
    expect(svg).toContain('height="200"');
  });
});

describe("page overview overlay positioning regression tests", () => {
  function createElement(overrides: Partial<PageOverviewElement>): PageOverviewElement {
    return {
      issueId: "test-issue",
      ruleId: "test-rule",
      severity: "moderate",
      selector: "#test",
      nodeIndex: 0,
      xPercent: 10,
      yPercent: 10,
      widthPercent: 10,
      heightPercent: 10,
      x: 100,
      y: 100,
      width: 100,
      height: 50,
      ...overrides,
    };
  }

  it("generates SVG rect at exact element coordinates", () => {
    const element = createElement({ x: 150, y: 200, width: 80, height: 40 });
    const svg = buildPageOverviewOverlaySvg([element], 1280, 720).toString("utf-8");

    expect(svg).toContain('x="150"');
    expect(svg).toContain('y="200"');
    expect(svg).toContain('width="80"');
    expect(svg).toContain('height="40"');
  });

  it("uses correct severity colors for each element", () => {
    const elements: PageOverviewElement[] = [
      createElement({ issueId: "crit", severity: "critical", x: 10, y: 10 }),
      createElement({ issueId: "ser", severity: "serious", x: 10, y: 100 }),
      createElement({ issueId: "mod", severity: "moderate", x: 10, y: 200 }),
      createElement({ issueId: "min", severity: "minor", x: 10, y: 300 }),
    ];

    const svg = buildPageOverviewOverlaySvg(elements, 1280, 720).toString("utf-8");

    expect(svg).toContain(SEVERITY_COLORS.critical.stroke);
    expect(svg).toContain(SEVERITY_COLORS.serious.stroke);
    expect(svg).toContain(SEVERITY_COLORS.moderate.stroke);
    expect(svg).toContain(SEVERITY_COLORS.minor.stroke);
  });

  it("rect coordinates match the element bounds exactly", () => {
    const testCases = [
      { x: 0, y: 0, width: 100, height: 50 },
      { x: 500, y: 300, width: 200, height: 100 },
      { x: 1000, y: 600, width: 150, height: 75 },
    ];

    for (const bounds of testCases) {
      const element = createElement(bounds);
      const svg = buildPageOverviewOverlaySvg([element], 1280, 720).toString("utf-8");

      expect(svg).toContain(`x="${bounds.x}"`);
      expect(svg).toContain(`y="${bounds.y}"`);
      expect(svg).toContain(`width="${bounds.width}"`);
      expect(svg).toContain(`height="${bounds.height}"`);
    }
  });

  it("multiple elements each get their own rect at correct positions", () => {
    const elements = [
      createElement({ issueId: "btn1", x: 100, y: 50, width: 80, height: 40 }),
      createElement({ issueId: "btn2", x: 200, y: 50, width: 80, height: 40 }),
      createElement({ issueId: "link", x: 150, y: 150, width: 120, height: 30 }),
    ];

    const svg = buildPageOverviewOverlaySvg(elements, 1280, 720).toString("utf-8");
    const rectMatches = svg.match(/<rect[^>]+>/g) ?? [];

    expect(rectMatches.length).toBe(3);

    // Verify each element's coordinates appear in the SVG
    expect(svg).toContain('x="100"');
    expect(svg).toContain('x="200"');
    expect(svg).toContain('x="150"');
    expect(svg).toContain('y="50"');
    expect(svg).toContain('y="150"');
  });

  it("SVG dimensions match the screenshot dimensions", () => {
    const element = createElement({});
    const screenshotWidth = 2560;
    const screenshotHeight = 3030;

    const svg = buildPageOverviewOverlaySvg(
      [element],
      screenshotWidth,
      screenshotHeight,
    ).toString("utf-8");

    expect(svg).toContain(`width="${screenshotWidth}"`);
    expect(svg).toContain(`height="${screenshotHeight}"`);
  });

  it("handles elements at edge of viewport", () => {
    const edgeElement = createElement({ x: 0, y: 0, width: 50, height: 30 });
    const svg = buildPageOverviewOverlaySvg([edgeElement], 1280, 720).toString("utf-8");

    expect(svg).toContain('x="0"');
    expect(svg).toContain('y="0"');
  });
});
