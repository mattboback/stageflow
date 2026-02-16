import { describe, expect, it } from "vitest";

import {
  getRuleCategory,
  getRuleTitle,
  hasRuleTitle,
} from "../../src/config/rule-titles";

describe("rule-titles", () => {
  it("returns curated titles for known rule ids", () => {
    expect(getRuleTitle("image-alt")).toBe("Image Missing Alternative Text");
    expect(hasRuleTitle("image-alt")).toBe(true);
    expect(getRuleCategory("image-alt")).toBe("Images & Media");
  });

  it("humanizes unknown rule ids", () => {
    expect(getRuleTitle("stageflow-custom_rule-id")).toBe("Stageflow Custom Rule Id");
    expect(getRuleTitle("foo_bar-baz")).toBe("Foo Bar Baz");
    expect(hasRuleTitle("not-a-rule")).toBe(false);
  });

  it("defaults unknown categories to Technical", () => {
    expect(getRuleCategory("not-a-rule")).toBe("Technical");
  });
});
