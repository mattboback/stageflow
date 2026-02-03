import type { IssueSeverity } from "../core/types";

export const SEVERITY_LEVELS = [
  "critical",
  "serious",
  "moderate",
  "minor",
  "info",
] as const;

const SEVERITY_RANK: Record<IssueSeverity, number> = {
  critical: 4,
  serious: 3,
  moderate: 2,
  minor: 1,
  info: 0,
};

export function normalizeSeverity(
  value: string | undefined | null,
  fallback: IssueSeverity = "info",
): IssueSeverity {
  if (!value) {
    return fallback;
  }

  const normalized = value.trim().toLowerCase();
  if ((SEVERITY_LEVELS as readonly string[]).includes(normalized)) {
    return normalized as IssueSeverity;
  }

  return fallback;
}

export function severityRank(severity: IssueSeverity): number {
  return SEVERITY_RANK[severity];
}

export function incrementSeverityCount(
  counts: Record<Exclude<IssueSeverity, "info">, number>,
  severity: IssueSeverity,
): void {
  if (severity === "info") {
    return;
  }
  counts[severity] += 1;
}
