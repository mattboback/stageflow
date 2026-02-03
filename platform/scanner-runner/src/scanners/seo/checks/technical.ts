import type { SEOCheck } from "../types";

export const TECHNICAL_CHECKS: SEOCheck[] = [
  {
    id: "missing-canonical",
    title: "Missing Canonical URL",
    severity: "moderate",
    category: "technical",
    helpUrl: "https://moz.com/learn/seo/canonicalization",
    check: (data) => {
      if (!data.canonical) {
        return {
          passed: false,
          message:
            "Page is missing a canonical URL. This helps prevent duplicate content issues.",
        };
      }

      return null;
    },
  },
  {
    id: "missing-viewport",
    title: "Missing Viewport Meta Tag",
    severity: "serious",
    category: "technical",
    check: (data) => {
      if (!data.viewport) {
        return {
          passed: false,
          message:
            "Page is missing viewport meta tag. This affects mobile usability and SEO.",
        };
      }

      return null;
    },
  },
  {
    id: "missing-language",
    title: "Missing Language Attribute",
    severity: "moderate",
    category: "technical",
    check: (data) => {
      if (!data.language) {
        return {
          passed: false,
          message:
            "HTML element is missing lang attribute. This helps search engines and screen readers.",
        };
      }

      return null;
    },
  },
  {
    id: "missing-structured-data",
    title: "Missing Structured Data",
    severity: "info",
    category: "technical",
    helpUrl:
      "https://developers.google.com/search/docs/appearance/structured-data/intro-structured-data",
    check: (data) => {
      if (data.structuredData.length === 0) {
        return {
          passed: false,
          message:
            "No structured data (JSON-LD) found. Structured data can improve rich snippets in search results.",
        };
      }

      return null;
    },
  },
];
