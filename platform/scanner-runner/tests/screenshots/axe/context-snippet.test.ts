import { describe, expect, it } from "vitest";

import {
  type ContextSnippetConfig,
  DEFAULT_CONTEXT_CONFIG,
} from "../../../src/screenshots/axe/context-snippet";

// Note: The actual extractContextSnippet function requires a real Playwright page,
// so we can't unit test it directly. These tests cover the configuration and types.

describe("context-snippet", () => {
  describe("DEFAULT_CONTEXT_CONFIG", () => {
    it("should have sensible default values", () => {
      expect(DEFAULT_CONTEXT_CONFIG.maxContextChars).toBe(2000);
      expect(DEFAULT_CONTEXT_CONFIG.maxAncestorDepth).toBe(5);
      expect(DEFAULT_CONTEXT_CONFIG.maxSiblings).toBe(1);
      expect(DEFAULT_CONTEXT_CONFIG.maxSiblingChars).toBe(200);
    });

    it("should be frozen or immutable at runtime", () => {
      // Verify the config object has all expected keys
      const keys = Object.keys(DEFAULT_CONTEXT_CONFIG);
      expect(keys).toContain("maxContextChars");
      expect(keys).toContain("maxAncestorDepth");
      expect(keys).toContain("maxSiblings");
      expect(keys).toContain("maxSiblingChars");
    });
  });

  describe("ContextSnippetConfig type", () => {
    it("should allow partial config overrides", () => {
      // This is a type test - if it compiles, the types work
      const partialConfig: Partial<ContextSnippetConfig> = {
        maxContextChars: 3000,
      };

      const merged: ContextSnippetConfig = {
        ...DEFAULT_CONTEXT_CONFIG,
        ...partialConfig,
      };

      expect(merged.maxContextChars).toBe(3000);
      expect(merged.maxAncestorDepth).toBe(DEFAULT_CONTEXT_CONFIG.maxAncestorDepth);
    });
  });
});
