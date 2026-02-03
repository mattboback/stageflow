import { describe, expect, it } from "vitest";

import type { AxeViolation } from "../../../src/screenshots/axe/types";

import {
  extractAxeViolationTargets,
  normalizeSelectors,
} from "../../../src/screenshots/axe/targets";

describe("normalizeSelectors", () => {
  it("trims and filters empty values", () => {
    expect(normalizeSelectors([" #a ", " ", "", "#b"])).toEqual(["#a", "#b"]);
  });

  it("stringifies non-strings safely", () => {
    expect(normalizeSelectors([123, false, null])).toEqual(["123", "false", "null"]);
  });
});

describe("extractAxeViolationTargets", () => {
  it("extracts de-duped targets from all nodes", () => {
    const v: AxeViolation = {
      id: "r1",
      nodes: [{ target: ["#a", " #b "] }, { target: ["#b", "#c"] }],
    };
    expect(extractAxeViolationTargets(v)).toEqual({
      targets: ["#a", "#b", "#c"],
      selector: "#a",
    });
  });

  it("handles missing nodes", () => {
    const v: AxeViolation = { id: "r1" };
    expect(extractAxeViolationTargets(v)).toEqual({ targets: [], selector: undefined });
  });
});
