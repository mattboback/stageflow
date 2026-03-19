import type { Page } from "playwright";

import type { AgentGoal, GoalStatus, SuccessCriterion } from "./types";

export async function checkGoal(page: Page, goal: AgentGoal): Promise<GoalStatus> {
  const criteria = goal.successCriteria ?? [];
  if (criteria.length === 0) {
    return {
      achieved: false,
      confidence: 0.2,
      reason: "No success criteria provided",
    };
  }

  for (const criterion of criteria) {
    const met = await checkCriterion(page, criterion);
    if (!met) {
      return {
        achieved: false,
        confidence: 0.6,
        reason: `Criterion not met: ${criterion.type}(${criterion.value})`,
      };
    }
  }

  return { achieved: true, confidence: 1, reason: "All success criteria met" };
}

async function checkCriterion(page: Page, criterion: SuccessCriterion): Promise<boolean> {
  const url = page.url();

  if (criterion.type === "url-contains") {
    return url.includes(criterion.value);
  }

  if (criterion.type === "url-matches") {
    try {
      return new RegExp(criterion.value).test(url);
    } catch {
      return false;
    }
  }

  if (criterion.type === "element-visible") {
    try {
      return await page.locator(criterion.value).first().isVisible();
    } catch {
      return false;
    }
  }

  if (criterion.type === "text-visible") {
    try {
      return await page.getByText(criterion.value).first().isVisible();
    } catch {
      return false;
    }
  }

  return false;
}
