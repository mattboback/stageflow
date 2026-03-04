import type {
  BoundingBox,
  FriendlyNodeInfo,
  LocationInfo,
  UnifiedReportV2,
  UserImpact,
} from "@stageflow/contracts-report";

import { createHash } from "node:crypto";
import path from "node:path";

import type {
  IssueSeverity,
  PageScanResult,
  Provenance,
  ScannerMetadata,
  ScanResults,
} from "./types";

import { normalizeSeverity } from "../utils/severity";
import { resultsPath } from "./artifact-paths";

type SeverityCounts = Record<IssueSeverity, number>;

interface PageOverview {
  screenshotFilename: string;
  pageWidth: number;
  pageHeight: number;
  elements: {
    issueId: string;
    ruleId: string;
    severity: IssueSeverity;
    selector: string;
    nodeIndex: number;
    xPercent: number;
    yPercent: number;
    widthPercent: number;
    heightPercent: number;
    x: number;
    y: number;
    width: number;
    height: number;
  }[];
}

export class WebServerFormatter {
  format(
    provenance: Provenance,
    results: ScanResults,
    metadata: ScannerMetadata,
  ): UnifiedReportV2 {
    const scannerId = metadata.name;
    const pages: UnifiedReportV2["pages"] = [];
    const issues: UnifiedReportV2["issues"] = [];
    const artifacts: NonNullable<UnifiedReportV2["artifacts"]> = [];

    const severityTotals: SeverityCounts = this.emptySeverityCounts();
    const baseUrl =
      typeof provenance.base_url === "string" && provenance.base_url.trim().length > 0
        ? provenance.base_url
        : undefined;

    for (const page of results.pages) {
      const pageSeverity: SeverityCounts = this.emptySeverityCounts();
      const rawResults = this.extractPageOverview(page);

      if (rawResults?.pageOverview?.screenshotFilename) {
        artifacts.push({
          id: `page-overview-${scannerId}-${page.pageId}`,
          type: "page-overview",
          path: `screenshots/${rawResults.pageOverview.screenshotFilename}`,
          mime: this.guessScreenshotMime(rawResults.pageOverview.screenshotFilename),
        });
      }

      for (const issue of page.issues) {
        const severity = normalizeSeverity(issue.severity, "info");
        pageSeverity[severity] += 1;
        severityTotals[severity] += 1;

        const issueMetadata = (issue.metadata ?? {}) as {
          impact?: unknown;
          thumbnail?: string;
          friendlyNode?: unknown;
          locationInfo?: unknown;
          userImpact?: unknown;
          nodes?: {
            target?: string[];
            html?: string;
            failureSummary?: string;
            selector?: string;
            textSnippet?: string;
            boundingBox?: unknown;
            contextHtml?: string;
            ancestorPath?: string;
          }[];
          nodeCount?: number;
          tags?: string[];
          ruleBehavior?: { tags?: string[] };
        };

        const nodes = Array.isArray(issueMetadata.nodes) ? issueMetadata.nodes : [];
        const nodeCount =
          typeof issueMetadata.nodeCount === "number"
            ? issueMetadata.nodeCount
            : Math.max(nodes.length, 1);
        const tags = this.normalizeStringArray(issueMetadata.tags);
        const wcagTags = this.extractWcagTags(tags, issueMetadata.ruleBehavior);

        const issueId = this.generateIssueFingerprint(
          scannerId,
          issue.id,
          page.pageId,
          nodes[0]?.selector ?? nodes[0]?.target?.[0] ?? "",
        );

        const scannerData = this.extractScannerData(issue.metadata);

        const occurrences: NonNullable<UnifiedReportV2["issues"][number]["occurrences"]> =
          nodes.slice(0, 5).map((node, idx) => ({
            target: node.target?.filter((t) => typeof t === "string"),
            selector: node.selector ?? node.target?.[0],
            html: node.html,
            contextHtml: node.contextHtml,
            ancestorPath: node.ancestorPath,
            failureSummary: node.failureSummary,
            textSnippet: node.textSnippet,
            boundingBox: this.asBoundingBox(node.boundingBox),
            elementId: `${issueId}-el-${idx}`,
          }));

        if (issue.screenshot) {
          const artifactId = `ss-${issueId}`;
          artifacts.push({
            id: artifactId,
            type: "screenshot",
            path: `screenshots/${issue.screenshot}`,
            mime: this.guessScreenshotMime(issue.screenshot),
          });
          for (const occurrence of occurrences) {
            occurrence.artifactIds = [artifactId];
          }
        }

        const severityRaw =
          typeof issueMetadata.impact === "string" ? issueMetadata.impact : undefined;

        const friendlyNode =
          typeof issueMetadata.friendlyNode === "object" &&
          issueMetadata.friendlyNode !== null
            ? (issueMetadata.friendlyNode as FriendlyNodeInfo)
            : undefined;

        const locationInfo =
          typeof issueMetadata.locationInfo === "object" &&
          issueMetadata.locationInfo !== null
            ? (issueMetadata.locationInfo as LocationInfo)
            : undefined;

        const userImpact =
          typeof issueMetadata.userImpact === "object" &&
          issueMetadata.userImpact !== null
            ? (issueMetadata.userImpact as UserImpact)
            : undefined;

        const helpUrl =
          typeof issue.helpUrl === "string" && issue.helpUrl.trim().length > 0
            ? issue.helpUrl
            : undefined;

        issues.push({
          id: issueId,
          scanner: scannerId,
          ruleId: issue.id,
          severity,
          severityRaw,
          title: issue.title,
          description: issue.description || issue.title,
          helpUrl,
          wcagTags,
          pageId: page.pageId,
          pageUrl: page.url,
          elementCount: nodeCount,
          elementsTruncated: nodes.length < nodeCount,
          occurrences,
          category: issue.category,
          friendlyNode,
          locationInfo,
          userImpact,
          howToFix: nodes[0]?.failureSummary,
          scannerData,
        });
      }

      pages.push({
        id: page.pageId,
        url: page.url,
        path: page.path,
        issueCount: page.issues.length,
        bySeverity: pageSeverity,
        durationMs: page.durationMs,
        startedAt: page.startedAt,
        finishedAt: page.finishedAt,
        pageOverview: rawResults?.pageOverview ?? undefined,
      });
    }

    return {
      version: "2.0.0",
      meta: {
        jobId: results.jobId,
        baseUrl,
        scannedAt: results.startedAt,
        completedAt: results.completedAt,
        durationMs: results.durationMs,
      },
      summary: {
        totalIssues: results.summary.totalIssues,
        bySeverity: severityTotals,
        byScanner: { [scannerId]: results.summary.totalIssues },
        pagesScanned: results.pages.length,
        pagesWithIssues: results.summary.pagesWithIssues,
        lighthouseCategories: results.summary.lighthouseCategories,
      },
      scanners: [
        {
          id: scannerId,
          name: scannerId,
          status: "success",
          issueCount: results.summary.totalIssues,
          severity: severityTotals,
          toolVersion: results.version,
          startedAt: results.startedAt,
          completedAt: results.completedAt,
          durationMs: results.durationMs,
          resultsPath: resultsPath(results.jobId, scannerId),
        },
      ],
      pages,
      issues,
      artifacts,
      errors: [],
    };
  }

  private emptySeverityCounts(): SeverityCounts {
    return {
      critical: 0,
      serious: 0,
      moderate: 0,
      minor: 0,
      info: 0,
    };
  }

  private generateIssueFingerprint(
    scanner: string,
    ruleId: string,
    pageId: string,
    selector: string,
  ): string {
    const data = `${scanner}|${ruleId}|${pageId}|${selector}`.toLowerCase();
    return createHash("sha256").update(data).digest("hex").slice(0, 12);
  }

  private extractWcagTags(tags: string[], ruleBehavior?: { tags?: string[] }): string[] {
    const allTags = [...tags, ...(ruleBehavior?.tags ?? [])];
    return allTags.filter(
      (tag) => /^wcag/i.test(tag) || /^level\\s+(aaa|aa|a)$/i.test(tag),
    );
  }

  private normalizeStringArray(value: unknown): string[] {
    if (!Array.isArray(value)) {
      return [];
    }
    return value.filter(
      (item): item is string => typeof item === "string" && item.trim().length > 0,
    );
  }

  private extractScannerData(
    metadata: unknown,
  ): Record<string, unknown> | undefined {
    if (typeof metadata !== "object" || metadata === null || Array.isArray(metadata)) {
      return undefined;
    }

    const omit = new Set([
      "impact",
      "thumbnail",
      "friendlyNode",
      "locationInfo",
      "userImpact",
      "nodes",
      "nodeCount",
      "tags",
      "ruleBehavior",
      "elementBounds",
    ]);

    const data: Record<string, unknown> = {};
    for (const [key, value] of Object.entries(metadata)) {
      if (omit.has(key)) {
        continue;
      }
      if (value === undefined) {
        continue;
      }
      data[key] = value;
    }

    return Object.keys(data).length > 0 ? data : undefined;
  }

  private extractPageOverview(
    page: PageScanResult,
  ): { pageOverview?: PageOverview } | null {
    const rawResults = page.rawResults as { pageOverview?: PageOverview } | null;
    return rawResults;
  }

  private guessScreenshotMime(filename: string): string {
    const ext = path.extname(filename).toLowerCase();
    if (ext === ".webp") {
      return "image/webp";
    }

    return "image/png";
  }

  private asBoundingBox(value: unknown): BoundingBox | undefined {
    if (typeof value !== "object" || value === null) {
      return undefined;
    }

    const maybe = value as Partial<BoundingBox>;
    if (
      typeof maybe.x === "number" &&
      typeof maybe.y === "number" &&
      typeof maybe.width === "number" &&
      typeof maybe.height === "number"
    ) {
      return { x: maybe.x, y: maybe.y, width: maybe.width, height: maybe.height };
    }

    return undefined;
  }
}
