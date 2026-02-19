import { describe, expect, it } from "vitest";

import type { PageSEOData } from "../../../src/scanners/seo/types";

import { SEO_CHECKS } from "../../../src/scanners/seo/checks";

function baseData(overrides: Partial<PageSEOData> = {}): PageSEOData {
  return {
    title: "A reasonably sized title for search engines",
    description:
      "A meta description that is long enough to satisfy typical search result snippets.",
    canonical: "https://example.com/page",
    robots: null,
    viewport: "width=device-width, initial-scale=1",
    charset: "utf-8",
    language: "en",
    headings: [{ level: 1, text: "Heading" }],
    images: [{ src: "https://example.com/a.png", alt: "Alt", width: 10, height: 10 }],
    links: [
      {
        href: "/internal",
        text: "Internal",
        isInternal: true,
        hasNofollow: false,
      },
    ],
    ogTags: {
      "og:title": "OG Title",
      "og:description": "OG Desc",
      "og:image": "https://example.com/og.png",
    },
    twitterTags: {},
    structuredData: [{}],
    wordCount: 500,
    url: "https://example.com/page",
    ...overrides,
  };
}

function getCheck(id: string) {
  const check = SEO_CHECKS.find((c) => c.id === id);
  if (!check) {
    throw new Error(`missing check: ${id}`);
  }

  return check;
}

describe("SEO checks", () => {
  // === META CHECKS ===
  describe("meta checks", () => {
    it("flags a missing title", () => {
      const result = getCheck("missing-title").check(baseData({ title: null }));
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
    });

    it("passes when title exists", () => {
      const result = getCheck("missing-title").check(baseData());
      expect(result).toBeNull();
    });

    it("flags title too short", () => {
      const result = getCheck("title-length").check(baseData({ title: "Short" }));
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
      expect(result?.message).toContain("too short");
    });

    it("flags title too long", () => {
      const result = getCheck("title-length").check(baseData({ title: "A".repeat(70) }));
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
      expect(result?.message).toContain("too long");
    });

    it("passes when title is optimal length", () => {
      const result = getCheck("title-length").check(baseData({ title: "A".repeat(55) }));
      expect(result).toBeNull();
    });

    it("flags missing meta description", () => {
      const result = getCheck("missing-description").check(
        baseData({ description: null }),
      );
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
    });

    it("flags description too short", () => {
      const result = getCheck("description-length").check(
        baseData({ description: "Too short" }),
      );
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
    });

    it("flags description too long", () => {
      const result = getCheck("description-length").check(
        baseData({ description: "A".repeat(180) }),
      );
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
    });
  });

  // === SOCIAL CHECKS ===
  describe("social checks", () => {
    it("flags missing og:image", () => {
      const result = getCheck("missing-og-tags").check(
        baseData({
          ogTags: {
            "og:title": "OG Title",
            "og:description": "OG Desc",
          },
        }),
      );
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
      expect(result?.message).toContain("og:image");
    });

    it("flags all missing og tags", () => {
      const result = getCheck("missing-og-tags").check(baseData({ ogTags: {} }));
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
      expect(result?.details?.missing).toContain("og:title");
      expect(result?.details?.missing).toContain("og:description");
      expect(result?.details?.missing).toContain("og:image");
    });

    it("passes when all required og tags present", () => {
      const result = getCheck("missing-og-tags").check(baseData());
      expect(result).toBeNull();
    });
  });

  // === TECHNICAL CHECKS ===
  describe("technical checks", () => {
    it("flags missing canonical URL", () => {
      const result = getCheck("missing-canonical").check(baseData({ canonical: null }));
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
    });

    it("passes when canonical exists", () => {
      const result = getCheck("missing-canonical").check(baseData());
      expect(result).toBeNull();
    });

    it("flags missing viewport meta tag", () => {
      const result = getCheck("missing-viewport").check(baseData({ viewport: null }));
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
    });

    it("flags missing language attribute", () => {
      const result = getCheck("missing-language").check(baseData({ language: null }));
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
    });

    it("flags missing structured data", () => {
      const result = getCheck("missing-structured-data").check(
        baseData({ structuredData: [] }),
      );
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
    });

    it("passes when structured data exists", () => {
      const result = getCheck("missing-structured-data").check(baseData());
      expect(result).toBeNull();
    });
  });

  // === IMAGE CHECKS ===
  describe("image checks", () => {
    it("flags images missing alt text", () => {
      const result = getCheck("images-missing-alt").check(
        baseData({
          images: [
            { src: "https://example.com/a.png", alt: "", width: 10, height: 10 },
            { src: "https://example.com/b.png", alt: null, width: 10, height: 10 },
          ],
        }),
      );
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
      expect(result?.message).toContain("2 image(s)");
    });

    it("passes when all images have alt text", () => {
      const result = getCheck("images-missing-alt").check(baseData());
      expect(result).toBeNull();
    });
  });

  // === HEADING CHECKS ===
  describe("heading checks", () => {
    it("flags missing H1 headings", () => {
      const result = getCheck("missing-h1").check(
        baseData({
          headings: [
            { level: 2, text: "H2" },
            { level: 3, text: "H3" },
          ],
        }),
      );
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
      expect(result?.message).toContain("missing an H1");
    });

    it("passes missing-h1 check when exactly one H1 exists", () => {
      const result = getCheck("missing-h1").check(
        baseData({
          headings: [{ level: 1, text: "Main heading" }],
        }),
      );
      expect(result).toBeNull();
    });

    it("flags multiple H1 headings and returns details", () => {
      const result = getCheck("multiple-h1").check(
        baseData({
          headings: [
            { level: 1, text: "Hero" },
            { level: 1, text: "Article title" },
          ],
        }),
      );
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
      expect(result?.details?.h1s).toEqual(["Hero", "Article title"]);
    });

    it("passes multiple-h1 check with a single H1", () => {
      const result = getCheck("multiple-h1").check(
        baseData({
          headings: [
            { level: 1, text: "H1" },
            { level: 2, text: "H2" },
          ],
        }),
      );
      expect(result).toBeNull();
    });

    it("passes heading hierarchy when levels are sequential", () => {
      const result = getCheck("heading-hierarchy").check(
        baseData({
          headings: [
            { level: 1, text: "H1" },
            { level: 2, text: "H2" },
            { level: 3, text: "H3" },
          ],
        }),
      );
      expect(result).toBeNull();
    });

    it("flags heading level skips", () => {
      const result = getCheck("heading-hierarchy").check(
        baseData({
          headings: [
            { level: 1, text: "H1" },
            { level: 3, text: "H3" },
          ],
        }),
      );
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
    });
  });

  // === CONTENT CHECKS ===
  describe("content checks", () => {
    it("flags thin content", () => {
      const result = getCheck("thin-content").check(baseData({ wordCount: 42 }));
      expect(result).not.toBeNull();
      expect(result?.passed).toBe(false);
    });

    it("passes with sufficient content", () => {
      const result = getCheck("thin-content").check(baseData());
      expect(result).toBeNull();
    });
  });
});
