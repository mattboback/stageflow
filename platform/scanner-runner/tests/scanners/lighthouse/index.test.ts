/**
 * Lighthouse Scanner Tests
 *
 * Tests for pure helper functions in the Lighthouse scanner.
 * The actual Lighthouse integration requires browser/CDP and is not unit-tested.
 */

import { describe, expect, it } from "vitest";

import { LighthouseScanner } from "../../../src/scanners/lighthouse";

const scanner = new LighthouseScanner();

// Helper to access private methods for testing
function callPrivateMethod(
  instance: LighthouseScanner,
  methodName: string,
  ...args: unknown[]
): unknown {
  const method = (instance as unknown as Record<string, (...args: unknown[]) => unknown>)[
    methodName
  ];
  if (!method) {
    throw new Error(`Unknown method: ${methodName}`);
  }

  return method.apply(instance, args);
}

describe("LighthouseScanner", () => {
  describe("metadata", () => {
    it("has correct scanner name", () => {
      expect(scanner.metadata.name).toBe("lighthouse");
    });

    it("has version", () => {
      expect(scanner.metadata.version).toBeDefined();
    });

    it("has description", () => {
      expect(scanner.metadata.description).toContain("Lighthouse");
    });
  });

  describe("mapScoreToSeverity", () => {
    it("returns 'info' for null score", () => {
      const result = callPrivateMethod(scanner, "mapScoreToSeverity", null) as string;
      expect(result).toBe("info");
    });

    it("returns 'critical' for score of 0", () => {
      const result = callPrivateMethod(scanner, "mapScoreToSeverity", 0) as string;
      expect(result).toBe("critical");
    });

    it("returns 'serious' for score below 0.5", () => {
      const result = callPrivateMethod(scanner, "mapScoreToSeverity", 0.3) as string;
      expect(result).toBe("serious");

      const result2 = callPrivateMethod(scanner, "mapScoreToSeverity", 0.49) as string;
      expect(result2).toBe("serious");
    });

    it("returns 'moderate' for score between 0.5 and 0.9", () => {
      const result = callPrivateMethod(scanner, "mapScoreToSeverity", 0.5) as string;
      expect(result).toBe("moderate");

      const result2 = callPrivateMethod(scanner, "mapScoreToSeverity", 0.89) as string;
      expect(result2).toBe("moderate");
    });

    it("returns 'minor' for score 0.9 or above", () => {
      const result = callPrivateMethod(scanner, "mapScoreToSeverity", 0.9) as string;
      expect(result).toBe("minor");

      const result2 = callPrivateMethod(scanner, "mapScoreToSeverity", 0.95) as string;
      expect(result2).toBe("minor");
    });
  });

  describe("getAuditCategory", () => {
    const mockCategories = {
      accessibility: {
        id: "accessibility",
        title: "Accessibility",
        score: 0.9,
        auditRefs: [
          { id: "color-contrast", weight: 3 },
          { id: "image-alt", weight: 2 },
        ],
      },
      performance: {
        id: "performance",
        title: "Performance",
        score: 0.8,
        auditRefs: [
          { id: "first-contentful-paint", weight: 10 },
          { id: "speed-index", weight: 10 },
        ],
      },
      seo: {
        id: "seo",
        title: "SEO",
        score: 0.85,
        auditRefs: [{ id: "document-title", weight: 2 }],
      },
    };

    it("returns correct category for accessibility audit", () => {
      const result = callPrivateMethod(
        scanner,
        "getAuditCategory",
        "color-contrast",
        mockCategories,
      ) as string | null;
      expect(result).toBe("accessibility");
    });

    it("returns correct category for performance audit", () => {
      const result = callPrivateMethod(
        scanner,
        "getAuditCategory",
        "first-contentful-paint",
        mockCategories,
      ) as string | null;
      expect(result).toBe("performance");
    });

    it("returns correct category for SEO audit", () => {
      const result = callPrivateMethod(
        scanner,
        "getAuditCategory",
        "document-title",
        mockCategories,
      ) as string | null;
      expect(result).toBe("seo");
    });

    it("returns null for unknown audit", () => {
      const result = callPrivateMethod(
        scanner,
        "getAuditCategory",
        "unknown-audit",
        mockCategories,
      ) as string | null;
      expect(result).toBeNull();
    });
  });

  describe("getHelpUrl", () => {
    it("returns specific URL for known accessibility audits", () => {
      const result = callPrivateMethod(scanner, "getHelpUrl", "color-contrast") as string;
      expect(result).toContain("color-contrast");
      expect(result).toContain("developer.chrome.com");
    });

    it("returns specific URL for image-alt audit", () => {
      const result = callPrivateMethod(scanner, "getHelpUrl", "image-alt") as string;
      expect(result).toContain("image-alt");
    });

    it("returns specific URL for button-name audit", () => {
      const result = callPrivateMethod(scanner, "getHelpUrl", "button-name") as string;
      expect(result).toContain("button-name");
    });

    it("returns overview URL for unknown audits", () => {
      const result = callPrivateMethod(scanner, "getHelpUrl", "unknown-audit") as string;
      expect(result).toContain("overview");
    });
  });

  describe("extractIssues", () => {
    const createMockLighthouseResult = (audits: Record<string, unknown>) => ({
      requestedUrl: "https://example.com",
      finalUrl: "https://example.com",
      fetchTime: new Date().toISOString(),
      categories: {
        accessibility: {
          id: "accessibility",
          title: "Accessibility",
          score: 0.8,
          auditRefs: Object.keys(audits).map((id) => ({ id, weight: 1 })),
        },
      },
      audits,
    });

    it("extracts issues from failing audits", () => {
      const lhResult = createMockLighthouseResult({
        "color-contrast": {
          id: "color-contrast",
          title: "Background and foreground colors have sufficient contrast ratio",
          description: "Low-contrast text is difficult to read.",
          score: 0.5,
          scoreDisplayMode: "numeric",
        },
      });

      const issues = callPrivateMethod(scanner, "extractIssues", lhResult) as {
        id: string;
        title: string;
        severity: string;
        category: string;
      }[];

      expect(issues).toHaveLength(1);
      expect(issues[0]!.id).toBe("color-contrast");
      expect(issues[0]!.title).toContain("contrast");
      expect(issues[0]!.severity).toBe("moderate");
      expect(issues[0]!.category).toBe("accessibility");
    });

    it("skips passing audits (score === 1)", () => {
      const lhResult = createMockLighthouseResult({
        "image-alt": {
          id: "image-alt",
          title: "Image elements have alt attributes",
          description: "Images need alt text.",
          score: 1,
          scoreDisplayMode: "binary",
        },
      });

      const issues = callPrivateMethod(scanner, "extractIssues", lhResult) as unknown[];
      expect(issues).toHaveLength(0);
    });

    it("skips audits with null score", () => {
      const lhResult = createMockLighthouseResult({
        "skip-link": {
          id: "skip-link",
          title: "Skip link",
          description: "Page has a skip link.",
          score: null,
          scoreDisplayMode: "notApplicable",
        },
      });

      const issues = callPrivateMethod(scanner, "extractIssues", lhResult) as unknown[];
      expect(issues).toHaveLength(0);
    });

    it("skips informative audits", () => {
      const lhResult = createMockLighthouseResult({
        "total-byte-weight": {
          id: "total-byte-weight",
          title: "Avoids enormous network payloads",
          description: "Network size info.",
          score: 0.5,
          scoreDisplayMode: "informative",
        },
      });

      const issues = callPrivateMethod(scanner, "extractIssues", lhResult) as unknown[];
      expect(issues).toHaveLength(0);
    });

    it("skips manual audits", () => {
      const lhResult = createMockLighthouseResult({
        "focus-traps": {
          id: "focus-traps",
          title: "Focus is not trapped in an interactive element",
          description: "Manual check required.",
          score: 0.5,
          scoreDisplayMode: "manual",
        },
      });

      const issues = callPrivateMethod(scanner, "extractIssues", lhResult) as unknown[];
      expect(issues).toHaveLength(0);
    });

    it("maps critical score (0) to critical severity", () => {
      const lhResult = createMockLighthouseResult({
        "html-has-lang": {
          id: "html-has-lang",
          title: "HTML element has a lang attribute",
          description: "No lang attribute found.",
          score: 0,
          scoreDisplayMode: "binary",
        },
      });

      const issues = callPrivateMethod(scanner, "extractIssues", lhResult) as {
        severity: string;
      }[];

      expect(issues).toHaveLength(1);
      expect(issues[0]!.severity).toBe("critical");
    });

    it("includes metadata from audit details", () => {
      const lhResult = createMockLighthouseResult({
        "color-contrast": {
          id: "color-contrast",
          title: "Color contrast",
          description: "Low contrast.",
          score: 0.3,
          scoreDisplayMode: "numeric",
          displayValue: "5 failing elements",
          numericValue: 5,
          details: {
            type: "table",
            items: [{ element: "<p>Low contrast</p>" }],
          },
        },
      });

      const issues = callPrivateMethod(scanner, "extractIssues", lhResult) as {
        metadata: Record<string, unknown>;
      }[];

      expect(issues).toHaveLength(1);
      expect(issues[0]!.metadata.score).toBe(0.3);
      expect(issues[0]!.metadata.displayValue).toBe("5 failing elements");
      expect(issues[0]!.metadata.numericValue).toBe(5);
      expect(issues[0]!.metadata.details).toBeDefined();
    });

    it("uses 'general' category when audit not in any category", () => {
      const lhResult = {
        requestedUrl: "https://example.com",
        finalUrl: "https://example.com",
        fetchTime: new Date().toISOString(),
        categories: {},
        audits: {
          "orphan-audit": {
            id: "orphan-audit",
            title: "Orphan audit",
            description: "Not in any category.",
            score: 0.5,
            scoreDisplayMode: "numeric",
          },
        },
      };

      const issues = callPrivateMethod(scanner, "extractIssues", lhResult) as {
        category: string;
      }[];

      expect(issues).toHaveLength(1);
      expect(issues[0]!.category).toBe("general");
    });

    it("handles multiple failing audits", () => {
      const lhResult = createMockLighthouseResult({
        "color-contrast": {
          id: "color-contrast",
          title: "Color contrast",
          description: "Low contrast.",
          score: 0.5,
          scoreDisplayMode: "numeric",
        },
        "image-alt": {
          id: "image-alt",
          title: "Image alt",
          description: "Missing alt.",
          score: 0.3,
          scoreDisplayMode: "binary",
        },
        "button-name": {
          id: "button-name",
          title: "Button name",
          description: "Missing name.",
          score: 0,
          scoreDisplayMode: "binary",
        },
      });

      const issues = callPrivateMethod(scanner, "extractIssues", lhResult) as unknown[];
      expect(issues).toHaveLength(3);
    });
  });

  describe("extractAuditNodes", () => {
    const createMockAudit = (items: Record<string, unknown>[]): {
      id: string;
      title: string;
      description: string;
      score: number;
      scoreDisplayMode: string;
      displayValue?: string;
      details?: { type: string; items: Record<string, unknown>[] };
    } => ({
      id: "test-audit",
      title: "Test Audit",
      description: "Test description",
      score: 0.5,
      scoreDisplayMode: "numeric",
      details: { type: "table", items },
    });

    it("returns empty array when no items in details", () => {
      const audit = createMockAudit([]);
      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as unknown[];
      expect(nodes).toEqual([]);
    });

    it("returns empty array when details is undefined", () => {
      const audit = {
        id: "test-audit",
        title: "Test Audit",
        description: "Test",
        score: 0.5,
        scoreDisplayMode: "numeric",
      };
      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as unknown[];
      expect(nodes).toEqual([]);
    });

    it("extracts selector from item.node.selector (accessibility audit format)", () => {
      const audit = createMockAudit([
        {
          node: {
            selector: "button.submit-btn",
            snippet: "<button class=\"submit-btn\">Submit</button>",
            path: "1,HTML,1,BODY,3,MAIN,1,FORM,2,BUTTON",
          },
        },
      ]);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
        target: string[];
        html?: string;
        ancestorPath?: string;
      }[];

      expect(nodes).toHaveLength(1);
      expect(nodes[0]!.selector).toBe("button.submit-btn");
      expect(nodes[0]!.target).toEqual(["button.submit-btn"]);
      expect(nodes[0]!.html).toBe("<button class=\"submit-btn\">Submit</button>");
      expect(nodes[0]!.ancestorPath).toBe("1,HTML,1,BODY,3,MAIN,1,FORM,2,BUTTON");
    });

    it("extracts selector from item.selector (direct selector format)", () => {
      const audit = createMockAudit([
        {
          selector: "#main-content",
          snippet: "<div id=\"main-content\"></div>",
        },
      ]);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
        target: string[];
      }[];

      expect(nodes).toHaveLength(1);
      expect(nodes[0]!.selector).toBe("#main-content");
      expect(nodes[0]!.target).toEqual(["#main-content"]);
    });

    it("extracts selector from item.element.selector", () => {
      const audit = createMockAudit([
        {
          element: {
            selector: "img.hero-image",
            snippet: "<img class=\"hero-image\" src=\"hero.jpg\">",
          },
        },
      ]);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
        html?: string;
      }[];

      expect(nodes).toHaveLength(1);
      expect(nodes[0]!.selector).toBe("img.hero-image");
      expect(nodes[0]!.html).toBe("<img class=\"hero-image\" src=\"hero.jpg\">");
    });

    it("extracts selector from item.source.selector", () => {
      const audit = createMockAudit([
        {
          source: {
            selector: "link[rel='stylesheet']",
            snippet: "<link rel=\"stylesheet\" href=\"styles.css\">",
          },
        },
      ]);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
      }[];

      expect(nodes).toHaveLength(1);
      expect(nodes[0]!.selector).toBe("link[rel='stylesheet']");
    });

    it("extracts selector from item.relatedNode.selector", () => {
      const audit = createMockAudit([
        {
          relatedNode: {
            selector: "label[for='email']",
            snippet: "<label for=\"email\">Email</label>",
          },
        },
      ]);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
      }[];

      expect(nodes).toHaveLength(1);
      expect(nodes[0]!.selector).toBe("label[for='email']");
    });

    it("prioritizes node.selector over other selector locations", () => {
      const audit = createMockAudit([
        {
          node: { selector: "node-selector" },
          selector: "direct-selector",
          element: { selector: "element-selector" },
          source: { selector: "source-selector" },
          relatedNode: { selector: "related-selector" },
        },
      ]);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
      }[];

      expect(nodes).toHaveLength(1);
      expect(nodes[0]!.selector).toBe("node-selector");
    });

    it("falls back through selector priority chain correctly", () => {
      // Test fallback to direct selector when node.selector is missing
      const audit1 = createMockAudit([
        {
          node: {}, // no selector
          selector: "direct-selector",
          element: { selector: "element-selector" },
        },
      ]);
      const nodes1 = callPrivateMethod(scanner, "extractAuditNodes", audit1) as {
        selector: string;
      }[];
      expect(nodes1[0]!.selector).toBe("direct-selector");

      // Test fallback to element.selector when node and direct are missing
      const audit2 = createMockAudit([
        {
          element: { selector: "element-selector" },
          source: { selector: "source-selector" },
        },
      ]);
      const nodes2 = callPrivateMethod(scanner, "extractAuditNodes", audit2) as {
        selector: string;
      }[];
      expect(nodes2[0]!.selector).toBe("element-selector");

      // Test fallback to source.selector
      const audit3 = createMockAudit([
        {
          source: { selector: "source-selector" },
          relatedNode: { selector: "related-selector" },
        },
      ]);
      const nodes3 = callPrivateMethod(scanner, "extractAuditNodes", audit3) as {
        selector: string;
      }[];
      expect(nodes3[0]!.selector).toBe("source-selector");

      // Test fallback to relatedNode.selector
      const audit4 = createMockAudit([
        {
          relatedNode: { selector: "related-selector" },
        },
      ]);
      const nodes4 = callPrivateMethod(scanner, "extractAuditNodes", audit4) as {
        selector: string;
      }[];
      expect(nodes4[0]!.selector).toBe("related-selector");
    });

    it("skips items with no selector in any location", () => {
      const audit = createMockAudit([
        { node: {}, element: {}, source: {} }, // no selectors
        { description: "Some other field" }, // no selectors
        { node: { selector: "valid-selector" } }, // has selector
      ]);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
      }[];

      expect(nodes).toHaveLength(1);
      expect(nodes[0]!.selector).toBe("valid-selector");
    });

    it("skips items with empty or whitespace-only selectors", () => {
      const audit = createMockAudit([
        { node: { selector: "" } },
        { node: { selector: "   " } },
        { selector: "" },
        { element: { selector: "   \n\t  " } },
        { node: { selector: "valid-selector" } },
      ]);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
      }[];

      expect(nodes).toHaveLength(1);
      expect(nodes[0]!.selector).toBe("valid-selector");
    });

    it("trims whitespace from selectors", () => {
      const audit = createMockAudit([
        { node: { selector: "  .padded-selector  " } },
      ]);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
      }[];

      expect(nodes).toHaveLength(1);
      expect(nodes[0]!.selector).toBe(".padded-selector");
    });

    it("deduplicates items with the same selector", () => {
      const audit = createMockAudit([
        { node: { selector: "#duplicate" } },
        { node: { selector: "#duplicate" } },
        { selector: "#duplicate" },
        { node: { selector: "#unique" } },
      ]);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
      }[];

      expect(nodes).toHaveLength(2);
      expect(nodes[0]!.selector).toBe("#duplicate");
      expect(nodes[1]!.selector).toBe("#unique");
    });

    it("limits results to 5 nodes maximum", () => {
      const items = Array.from({ length: 10 }, (_, i) => ({
        node: { selector: `#element-${i}` },
      }));
      const audit = createMockAudit(items);

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
      }[];

      expect(nodes).toHaveLength(5);
      expect(nodes[4]!.selector).toBe("#element-4");
    });

    it("extracts HTML snippet from multiple locations", () => {
      // From node.snippet
      const audit1 = createMockAudit([
        { node: { selector: "#a", snippet: "<div>Node snippet</div>" } },
      ]);
      const nodes1 = callPrivateMethod(scanner, "extractAuditNodes", audit1) as {
        html?: string;
      }[];
      expect(nodes1[0]!.html).toBe("<div>Node snippet</div>");

      // From item.snippet when node.snippet is missing
      const audit2 = createMockAudit([
        { node: { selector: "#b" }, snippet: "<div>Item snippet</div>" },
      ]);
      const nodes2 = callPrivateMethod(scanner, "extractAuditNodes", audit2) as {
        html?: string;
      }[];
      expect(nodes2[0]!.html).toBe("<div>Item snippet</div>");

      // From element.snippet as fallback
      const audit3 = createMockAudit([
        {
          selector: "#c",
          element: { snippet: "<div>Element snippet</div>" },
        },
      ]);
      const nodes3 = callPrivateMethod(scanner, "extractAuditNodes", audit3) as {
        html?: string;
      }[];
      expect(nodes3[0]!.html).toBe("<div>Element snippet</div>");
    });

    it("extracts ancestor path from node.path or item.path", () => {
      // From node.path
      const audit1 = createMockAudit([
        { node: { selector: "#a", path: "1,HTML,1,BODY,0,DIV" } },
      ]);
      const nodes1 = callPrivateMethod(scanner, "extractAuditNodes", audit1) as {
        ancestorPath?: string;
      }[];
      expect(nodes1[0]!.ancestorPath).toBe("1,HTML,1,BODY,0,DIV");

      // From item.path when node.path is missing
      const audit2 = createMockAudit([
        { node: { selector: "#b" }, path: "1,HTML,1,BODY,1,MAIN" },
      ]);
      const nodes2 = callPrivateMethod(scanner, "extractAuditNodes", audit2) as {
        ancestorPath?: string;
      }[];
      expect(nodes2[0]!.ancestorPath).toBe("1,HTML,1,BODY,1,MAIN");
    });

    it("extracts failure summary from item.explanation or item.displayValue", () => {
      // From item.explanation
      const audit1 = createMockAudit([
        {
          node: { selector: "#a" },
          explanation: "Element has insufficient contrast ratio",
        },
      ]);
      const nodes1 = callPrivateMethod(scanner, "extractAuditNodes", audit1) as {
        failureSummary?: string;
      }[];
      expect(nodes1[0]!.failureSummary).toBe("Element has insufficient contrast ratio");

      // From item.displayValue when explanation is missing
      const audit2 = createMockAudit([
        {
          node: { selector: "#b" },
          displayValue: "Contrast ratio: 2.5:1",
        },
      ]);
      const nodes2 = callPrivateMethod(scanner, "extractAuditNodes", audit2) as {
        failureSummary?: string;
      }[];
      expect(nodes2[0]!.failureSummary).toBe("Contrast ratio: 2.5:1");
    });

    it("falls back to audit.displayValue for failure summary", () => {
      const audit = {
        id: "test-audit",
        title: "Test Audit",
        description: "Test description",
        score: 0.5,
        scoreDisplayMode: "numeric",
        displayValue: "5 elements fail this audit",
        details: {
          type: "table",
          items: [{ node: { selector: "#a" } }],
        },
      };

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        failureSummary?: string;
      }[];

      expect(nodes[0]!.failureSummary).toBe("5 elements fail this audit");
    });

    it("skips null or non-object items", () => {
      const audit = {
        id: "test-audit",
        title: "Test Audit",
        description: "Test description",
        score: 0.5,
        scoreDisplayMode: "numeric",
        details: {
          type: "table",
          items: [
            null,
            undefined,
            "string-item",
            123,
            true,
            { node: { selector: "valid" } },
          ],
        },
      };

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
      }[];

      expect(nodes).toHaveLength(1);
      expect(nodes[0]!.selector).toBe("valid");
    });

    it("handles real-world Lighthouse accessibility audit structure", () => {
      const audit = {
        id: "color-contrast",
        title: "Background and foreground colors have sufficient contrast ratio",
        description: "Low-contrast text is difficult to read",
        score: 0.5,
        scoreDisplayMode: "numeric",
        displayValue: "3 failing elements",
        details: {
          type: "table",
          items: [
            {
              node: {
                selector: ".low-contrast-text",
                snippet: "<p class=\"low-contrast-text\">Hard to read</p>",
                path: "1,HTML,1,BODY,0,MAIN,2,P",
              },
            },
            {
              node: {
                selector: ".faded-link",
                snippet: "<a class=\"faded-link\" href=\"/\">Link</a>",
                path: "1,HTML,1,BODY,0,MAIN,3,A",
              },
            },
          ],
        },
      };

      const nodes = callPrivateMethod(scanner, "extractAuditNodes", audit) as {
        selector: string;
        target: string[];
        html?: string;
        ancestorPath?: string;
        failureSummary?: string;
      }[];

      expect(nodes).toHaveLength(2);
      expect(nodes[0]).toEqual({
        selector: ".low-contrast-text",
        target: [".low-contrast-text"],
        html: "<p class=\"low-contrast-text\">Hard to read</p>",
        ancestorPath: "1,HTML,1,BODY,0,MAIN,2,P",
        failureSummary: "3 failing elements",
      });
      expect(nodes[1]).toEqual({
        selector: ".faded-link",
        target: [".faded-link"],
        html: "<a class=\"faded-link\" href=\"/\">Link</a>",
        ancestorPath: "1,HTML,1,BODY,0,MAIN,3,A",
        failureSummary: "3 failing elements",
      });
    });
  });
});
