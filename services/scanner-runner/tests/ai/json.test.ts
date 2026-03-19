import { describe, expect, it } from "vitest";

import { isPlainObject, parseFirstJsonObject } from "../../src/ai/json";

describe("isPlainObject", () => {
  it("returns true for plain objects", () => {
    expect(isPlainObject({})).toBe(true);
    expect(isPlainObject({ key: "value" })).toBe(true);
    expect(isPlainObject({ nested: { obj: true } })).toBe(true);
  });

  it("returns false for null", () => {
    expect(isPlainObject(null)).toBe(false);
  });

  it("returns false for undefined", () => {
    expect(isPlainObject(undefined)).toBe(false);
  });

  it("returns false for arrays", () => {
    expect(isPlainObject([])).toBe(false);
    expect(isPlainObject([1, 2, 3])).toBe(false);
    expect(isPlainObject([{ obj: true }])).toBe(false);
  });

  it("returns false for primitives", () => {
    expect(isPlainObject("string")).toBe(false);
    expect(isPlainObject(123)).toBe(false);
    expect(isPlainObject(true)).toBe(false);
    expect(isPlainObject(false)).toBe(false);
  });

  it("returns false for functions", () => {
    expect(isPlainObject(() => undefined)).toBe(false);
    expect(isPlainObject(() => undefined)).toBe(false);
  });
});

describe("parseFirstJsonObject", () => {
  describe("direct JSON parsing", () => {
    it("parses a simple JSON object", () => {
      const result = parseFirstJsonObject('{"action": "click", "selector": "#btn"}');
      expect(result).toEqual({ action: "click", selector: "#btn" });
    });

    it("parses nested JSON objects", () => {
      const result = parseFirstJsonObject(
        '{"action": {"type": "click"}, "confidence": 0.9}',
      );
      expect(result).toEqual({ action: { type: "click" }, confidence: 0.9 });
    });

    it("returns undefined for empty string", () => {
      expect(parseFirstJsonObject("")).toBeUndefined();
    });

    it("returns undefined for whitespace only", () => {
      expect(parseFirstJsonObject("   ")).toBeUndefined();
      expect(parseFirstJsonObject("\n\t")).toBeUndefined();
    });

    it("returns undefined for JSON arrays", () => {
      expect(parseFirstJsonObject("[1, 2, 3]")).toBeUndefined();
    });

    it("returns undefined for JSON primitives", () => {
      expect(parseFirstJsonObject('"string"')).toBeUndefined();
      expect(parseFirstJsonObject("123")).toBeUndefined();
      expect(parseFirstJsonObject("true")).toBeUndefined();
      expect(parseFirstJsonObject("null")).toBeUndefined();
    });
  });

  describe("fenced code block parsing", () => {
    it("extracts JSON from a json fenced block", () => {
      const input = `Here's the action:
\`\`\`json
{"action": "click", "selector": "#submit"}
\`\`\``;
      const result = parseFirstJsonObject(input);
      expect(result).toEqual({ action: "click", selector: "#submit" });
    });

    it("extracts JSON from unfenced block (no language)", () => {
      const input = `\`\`\`
{"type": "fill", "value": "test"}
\`\`\``;
      const result = parseFirstJsonObject(input);
      expect(result).toEqual({ type: "fill", value: "test" });
    });

    it("handles whitespace in fenced blocks", () => {
      const input = `\`\`\`json
  { "key": "value" }
\`\`\``;
      const result = parseFirstJsonObject(input);
      expect(result).toEqual({ key: "value" });
    });

    it("is case insensitive for JSON language tag", () => {
      const input = `\`\`\`JSON
{"test": true}
\`\`\``;
      const result = parseFirstJsonObject(input);
      expect(result).toEqual({ test: true });
    });
  });

  describe("brace extraction fallback", () => {
    it("extracts JSON embedded in prose", () => {
      const input = `Based on the page, I recommend: {"action": "click", "reasoning": "button visible"} which will progress the form.`;
      const result = parseFirstJsonObject(input);
      expect(result).toEqual({ action: "click", reasoning: "button visible" });
    });

    it("handles multiline JSON in prose", () => {
      const input = `Here's my decision:
{
  "action": "scroll",
  "direction": "down"
}
That should reveal more content.`;
      const result = parseFirstJsonObject(input);
      expect(result).toEqual({ action: "scroll", direction: "down" });
    });

    it("returns undefined when no braces found", () => {
      expect(parseFirstJsonObject("No JSON here")).toBeUndefined();
    });

    it("returns undefined when only opening brace found", () => {
      expect(parseFirstJsonObject("Incomplete { object")).toBeUndefined();
    });

    it("returns undefined when closing brace comes before opening", () => {
      expect(parseFirstJsonObject("} before { doesn't work")).toBeUndefined();
    });

    it("returns undefined for malformed JSON between braces", () => {
      expect(parseFirstJsonObject("{ not: valid json }")).toBeUndefined();
    });
  });

  describe("precedence", () => {
    it("prefers direct parsing over fenced extraction", () => {
      // If the whole string is valid JSON, use it directly
      const result = parseFirstJsonObject('{"direct": true}');
      expect(result).toEqual({ direct: true });
    });

    it("falls back to fenced block when direct parse fails", () => {
      const input = `Some text \`\`\`json
{"fenced": true}
\`\`\``;
      const result = parseFirstJsonObject(input);
      expect(result).toEqual({ fenced: true });
    });

    it("falls back to brace extraction when fenced block fails", () => {
      const input = `Text with {"extracted": true} in the middle`;
      const result = parseFirstJsonObject(input);
      expect(result).toEqual({ extracted: true });
    });
  });

  describe("edge cases", () => {
    it("handles complex nested objects", () => {
      const complex = {
        action: {
          type: "fill",
          selector: 'input[name="email"]',
          value: "test@example.com",
        },
        reasoning: "Filling email field",
        confidence: 0.95,
        alternatives: [{ type: "click", selector: "#skip" }],
      };
      const result = parseFirstJsonObject(JSON.stringify(complex));
      expect(result).toEqual(complex);
    });

    it("handles JSON with special characters", () => {
      const result = parseFirstJsonObject(
        '{"selector": "button:has-text(\\"Submit\\")"}',
      );
      expect(result).toEqual({ selector: 'button:has-text("Submit")' });
    });

    it("handles unicode in JSON", () => {
      const result = parseFirstJsonObject('{"text": "Hello \\u4e16\\u754c"}');
      expect(result?.text).toBe("Hello \u4e16\u754c");
    });

    it("trims whitespace before parsing", () => {
      const result = parseFirstJsonObject('  \n  {"trimmed": true}  \n  ');
      expect(result).toEqual({ trimmed: true });
    });
  });
});
