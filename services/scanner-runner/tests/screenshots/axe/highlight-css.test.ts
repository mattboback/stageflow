import { describe, expect, it } from "vitest";

import { buildHighlightCSSRules } from "../../../src/screenshots/axe/highlight-css";

describe("buildHighlightCSSRules", () => {
  it("returns null for empty selectors", () => {
    expect(buildHighlightCSSRules([], { highlightStyle: "solid" })).toBeNull();
  });

  it("builds dashed highlight rules", () => {
    const css = buildHighlightCSSRules(["#a"], { highlightStyle: "dashed" });
    expect(css).toContain("#a");
    expect(css).toContain("4px dashed");
    expect(css).toContain("rgba(255,45,85,0.1)");
  });

  it("builds solid highlight rules", () => {
    const css = buildHighlightCSSRules(["#a"], { highlightStyle: "solid" });
    expect(css).toContain("#a");
    expect(css).toContain("5px solid");
    expect(css).toContain("rgba(255, 0, 0, 0.1)");
  });
});
