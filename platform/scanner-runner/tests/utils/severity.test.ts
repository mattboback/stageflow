import { describe, expect, it } from "vitest";

import {
  incrementSeverityCount,
  normalizeSeverity,
  SEVERITY_LEVELS,
  severityRank,
} from "../../src/utils/severity";

describe("SEVERITY_LEVELS", () => {
  it("contains all expected severity levels", () => {
    expect(SEVERITY_LEVELS).toContain("critical");
    expect(SEVERITY_LEVELS).toContain("serious");
    expect(SEVERITY_LEVELS).toContain("moderate");
    expect(SEVERITY_LEVELS).toContain("minor");
    expect(SEVERITY_LEVELS).toContain("info");
  });

  it("has levels in order from most to least severe", () => {
    expect(SEVERITY_LEVELS[0]).toBe("critical");
    expect(SEVERITY_LEVELS[4]).toBe("info");
  });
});

describe("normalizeSeverity", () => {
  it("normalizes uppercase to lowercase", () => {
    expect(normalizeSeverity("CRITICAL")).toBe("critical");
    expect(normalizeSeverity("SERIOUS")).toBe("serious");
    expect(normalizeSeverity("MODERATE")).toBe("moderate");
    expect(normalizeSeverity("MINOR")).toBe("minor");
    expect(normalizeSeverity("INFO")).toBe("info");
  });

  it("trims whitespace", () => {
    expect(normalizeSeverity("  moderate  ")).toBe("moderate");
    expect(normalizeSeverity("\tcritical\n")).toBe("critical");
  });

  it("handles mixed case", () => {
    expect(normalizeSeverity("CrItIcAl")).toBe("critical");
    expect(normalizeSeverity("Serious")).toBe("serious");
  });

  it("returns fallback for invalid values", () => {
    expect(normalizeSeverity("unknown", "minor")).toBe("minor");
    expect(normalizeSeverity("invalid")).toBe("info"); // default fallback
    expect(normalizeSeverity("warning", "serious")).toBe("serious");
  });

  it("returns fallback for null/undefined", () => {
    expect(normalizeSeverity(null)).toBe("info");
    expect(normalizeSeverity(undefined)).toBe("info");
    expect(normalizeSeverity(null, "critical")).toBe("critical");
  });

  it("returns fallback for empty string", () => {
    expect(normalizeSeverity("")).toBe("info");
    expect(normalizeSeverity("   ")).toBe("info");
  });
});

describe("severityRank", () => {
  it("ranks severities by impact", () => {
    expect(severityRank("critical")).toBeGreaterThan(severityRank("serious"));
    expect(severityRank("serious")).toBeGreaterThan(severityRank("moderate"));
    expect(severityRank("moderate")).toBeGreaterThan(severityRank("minor"));
    expect(severityRank("minor")).toBeGreaterThan(severityRank("info"));
  });

  it("returns specific rank values", () => {
    expect(severityRank("critical")).toBe(4);
    expect(severityRank("serious")).toBe(3);
    expect(severityRank("moderate")).toBe(2);
    expect(severityRank("minor")).toBe(1);
    expect(severityRank("info")).toBe(0);
  });
});

describe("incrementSeverityCount", () => {
  it("increments critical count", () => {
    const counts = { critical: 0, serious: 0, moderate: 0, minor: 0 };
    incrementSeverityCount(counts, "critical");
    expect(counts.critical).toBe(1);
  });

  it("increments serious count", () => {
    const counts = { critical: 0, serious: 0, moderate: 0, minor: 0 };
    incrementSeverityCount(counts, "serious");
    expect(counts.serious).toBe(1);
  });

  it("increments moderate count", () => {
    const counts = { critical: 0, serious: 0, moderate: 0, minor: 0 };
    incrementSeverityCount(counts, "moderate");
    expect(counts.moderate).toBe(1);
  });

  it("increments minor count", () => {
    const counts = { critical: 0, serious: 0, moderate: 0, minor: 0 };
    incrementSeverityCount(counts, "minor");
    expect(counts.minor).toBe(1);
  });

  it("skips info severity", () => {
    const counts = { critical: 0, serious: 0, moderate: 0, minor: 0 };
    incrementSeverityCount(counts, "info");
    expect(counts).toEqual({ critical: 0, serious: 0, moderate: 0, minor: 0 });
  });

  it("accumulates multiple increments", () => {
    const counts = { critical: 0, serious: 0, moderate: 0, minor: 0 };
    incrementSeverityCount(counts, "critical");
    incrementSeverityCount(counts, "critical");
    incrementSeverityCount(counts, "minor");
    incrementSeverityCount(counts, "info");
    incrementSeverityCount(counts, "serious");

    expect(counts).toEqual({ critical: 2, serious: 1, moderate: 0, minor: 1 });
  });
});
