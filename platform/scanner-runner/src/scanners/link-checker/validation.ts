/**
 * Link Validation Utilities
 *
 * Pure functions for link checking and result processing.
 */

import type { IssueSeverity } from "../../core/types";
import type { LinkCheckResult } from "./types";

const REQUEST_TIMEOUT = 10000;
const USER_AGENT = "Stageflow-LinkChecker/1.0";

/**
 * Groups link check results by HTTP status code.
 */
export function groupByStatus(
  links: LinkCheckResult[],
): Record<string, LinkCheckResult[]> {
  const grouped: Record<string, LinkCheckResult[]> = {};
  for (const link of links) {
    const status = String(link.status ?? 0);
    grouped[status] ??= [];
    grouped[status].push(link);
  }
  return grouped;
}

/**
 * Maps HTTP status code to issue severity.
 */
export function getSeverityForStatus(status: number): IssueSeverity {
  if (status === 0) {
    return "serious";
  }
  if (status === 404) {
    return "serious";
  }
  if (status >= 500) {
    return "critical";
  }
  if (status >= 400) {
    return "moderate";
  }
  return "minor";
}

/**
 * Checks a single URL for availability, using HEAD with GET fallback.
 */
export async function checkSingleLink(url: string): Promise<LinkCheckResult> {
  const startTime = Date.now();
  const redirects: string[] = [];

  try {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => {
      controller.abort();
    }, REQUEST_TIMEOUT);

    const response = await fetch(url, {
      method: "HEAD",
      redirect: "follow",
      signal: controller.signal,
      headers: {
        "User-Agent": USER_AGENT,
      },
    });

    clearTimeout(timeoutId);

    if (response.redirected) {
      redirects.push(response.url);
    }

    return {
      url,
      status: response.status,
      error: null,
      redirects,
      responseTime: Date.now() - startTime,
    };
  } catch (_error) {
    // If HEAD fails, try GET (some servers don't support HEAD)
    try {
      const controller = new AbortController();
      const timeoutId = setTimeout(() => {
        controller.abort();
      }, REQUEST_TIMEOUT);

      const response = await fetch(url, {
        method: "GET",
        redirect: "follow",
        signal: controller.signal,
        headers: {
          "User-Agent": USER_AGENT,
        },
      });

      clearTimeout(timeoutId);

      return {
        url,
        status: response.status,
        error: null,
        redirects: response.redirected ? [response.url] : [],
        responseTime: Date.now() - startTime,
      };
    } catch (getError) {
      return {
        url,
        status: null,
        error: getError instanceof Error ? getError.message : "Connection failed",
        redirects: [],
        responseTime: Date.now() - startTime,
      };
    }
  }
}
