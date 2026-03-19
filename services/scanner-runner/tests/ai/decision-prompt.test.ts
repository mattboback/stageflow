import { describe, expect, it } from "vitest";

import type { AgentGoal, AgentStep, PagePerception } from "../../src/ai/types";

import { buildDecisionPrompt } from "../../src/ai/decision-prompt";

describe("buildDecisionPrompt", () => {
  const createPerception = (overrides: Partial<PagePerception> = {}): PagePerception => ({
    url: "https://example.com/login",
    title: "Login Page",
    pageType: "form",
    description: "A login form with email and password fields",
    interactiveElements: [],
    ...overrides,
  });

  const createGoal = (overrides: Partial<AgentGoal> = {}): AgentGoal => ({
    objective: "Log into the application",
    ...overrides,
  });

  const createStep = (overrides: Partial<AgentStep> = {}): AgentStep => ({
    stepNumber: 1,
    url: "https://example.com",
    action: { type: "click", selector: "#btn" },
    reasoning: "Click the button",
    success: true,
    durationMs: 100,
    ...overrides,
  });

  describe("basic structure", () => {
    it("includes the goal objective", () => {
      const perception = createPerception();
      const goal = createGoal({ objective: "Complete the checkout process" });

      const prompt = buildDecisionPrompt(perception, goal, []);

      expect(prompt).toContain("Goal: Complete the checkout process");
    });

    it("includes current URL and title", () => {
      const perception = createPerception({
        url: "https://shop.example.com/cart",
        title: "Shopping Cart",
      });

      const prompt = buildDecisionPrompt(perception, createGoal(), []);

      expect(prompt).toContain("Current URL: https://shop.example.com/cart");
      expect(prompt).toContain("Title: Shopping Cart");
    });

    it("includes page type and description", () => {
      const perception = createPerception({
        pageType: "checkout",
        description: "Final step of purchase process",
      });

      const prompt = buildDecisionPrompt(perception, createGoal(), []);

      expect(prompt).toContain("Page type: checkout");
      expect(prompt).toContain("Description: Final step of purchase process");
    });

    it("includes JSON schema instructions", () => {
      const prompt = buildDecisionPrompt(createPerception(), createGoal(), []);

      expect(prompt).toContain("Return ONLY JSON with this shape:");
      expect(prompt).toContain('"action":');
      expect(prompt).toContain(
        '"type": "click|fill|select|hover|scroll|keyboard|wait|done|stuck"',
      );
      expect(prompt).toContain('"reasoning":');
      expect(prompt).toContain('"confidence":');
    });
  });

  describe("input policy", () => {
    it("restricts input when inputValues provided", () => {
      const goal = createGoal({
        inputValues: {
          email: "test@example.com",
          password: "secret123",
        },
      });

      const prompt = buildDecisionPrompt(createPerception(), goal, []);

      expect(prompt).toContain(
        "You may only fill/select using one of these valueKey values:",
      );
      expect(prompt).toContain("email");
      expect(prompt).toContain("password");
    });

    it("disallows fill/select when no inputValues", () => {
      const goal = createGoal({ inputValues: undefined });

      const prompt = buildDecisionPrompt(createPerception(), goal, []);

      expect(prompt).toContain("Do not use fill/select unless explicitly allowed");
    });

    it("limits displayed input keys to 20", () => {
      const manyInputs: Record<string, string> = {};
      for (let i = 0; i < 30; i++) {
        manyInputs[`field${i}`] = `value${i}`;
      }
      const goal = createGoal({ inputValues: manyInputs });

      const prompt = buildDecisionPrompt(createPerception(), goal, []);

      // Should only include first 20 keys
      const matches = prompt.match(/field\d+/g) ?? [];
      expect(matches.length).toBeLessThanOrEqual(20);
    });
  });

  describe("history handling", () => {
    it("shows (none) when no history", () => {
      const prompt = buildDecisionPrompt(createPerception(), createGoal(), []);

      expect(prompt).toContain("Recent steps:");
      expect(prompt).toContain("(none)");
    });

    it("includes recent steps in history", () => {
      const steps: AgentStep[] = [
        createStep({
          stepNumber: 1,
          url: "https://a.com",
          action: { type: "click", selector: "#a" },
        }),
        createStep({
          stepNumber: 2,
          url: "https://b.com",
          action: { type: "fill", selector: "#b", value: "email" },
        }),
      ];

      const prompt = buildDecisionPrompt(createPerception(), createGoal(), steps);

      expect(prompt).toContain("1. https://a.com -> click");
      expect(prompt).toContain("2. https://b.com -> fill");
    });

    it("limits history to last 5 steps", () => {
      const steps: AgentStep[] = [];
      for (let i = 1; i <= 10; i++) {
        steps.push(
          createStep({
            stepNumber: i,
            url: `https://step${i}.com`,
            action: { type: "click", selector: `#btn${i}` },
          }),
        );
      }

      const prompt = buildDecisionPrompt(createPerception(), createGoal(), steps);

      // Should not contain early steps
      expect(prompt).not.toContain("https://step1.com");
      expect(prompt).not.toContain("https://step5.com");
      // Should contain recent steps
      expect(prompt).toContain("https://step6.com");
      expect(prompt).toContain("https://step10.com");
    });
  });

  describe("interactive elements", () => {
    it("shows (none) when no interactive elements", () => {
      const perception = createPerception({ interactiveElements: [] });

      const prompt = buildDecisionPrompt(perception, createGoal(), []);

      expect(prompt).toContain("Interactive elements (selectors):");
      expect(prompt).toContain("(none)");
    });

    it("lists interactive elements with selector and accessible name", () => {
      const perception = createPerception({
        interactiveElements: [
          {
            selector: "#login-btn",
            tagName: "button",
            accessibleName: "Log In",
            isVisible: true,
            isEnabled: true,
          },
          {
            selector: 'input[name="email"]',
            tagName: "input",
            text: "Email address",
            isVisible: true,
            isEnabled: true,
          },
        ],
      });

      const prompt = buildDecisionPrompt(perception, createGoal(), []);

      expect(prompt).toContain("1. #login-btn (Log In)");
      expect(prompt).toContain('2. input[name="email"] (Email address)');
    });

    it("falls back to tagName when no accessible name or text", () => {
      const perception = createPerception({
        interactiveElements: [
          {
            selector: ".mystery-div",
            tagName: "div",
            isVisible: true,
            isEnabled: true,
          },
        ],
      });

      const prompt = buildDecisionPrompt(perception, createGoal(), []);

      expect(prompt).toContain("1. .mystery-div (div)");
    });

    it("limits interactive elements to 25", () => {
      const elements = Array.from({ length: 40 }, (_, i) => ({
        selector: `#el${i}`,
        tagName: "button",
        accessibleName: `Button ${i}`,
        isVisible: true,
        isEnabled: true,
      }));
      const perception = createPerception({ interactiveElements: elements });

      const prompt = buildDecisionPrompt(perception, createGoal(), []);

      // Should include first 25
      expect(prompt).toContain("25. #el24");
      // Should not include element 26+
      expect(prompt).not.toContain("#el25");
      expect(prompt).not.toContain("#el39");
    });

    it("prioritizes accessibleName over text", () => {
      const perception = createPerception({
        interactiveElements: [
          {
            selector: "#btn",
            tagName: "button",
            accessibleName: "Submit Form",
            text: "Submit",
            isVisible: true,
            isEnabled: true,
          },
        ],
      });

      const prompt = buildDecisionPrompt(perception, createGoal(), []);

      expect(prompt).toContain("1. #btn (Submit Form)");
      expect(prompt).not.toContain("(Submit)");
    });
  });

  describe("action types", () => {
    it("documents all action types in schema", () => {
      const prompt = buildDecisionPrompt(createPerception(), createGoal(), []);

      expect(prompt).toContain("click");
      expect(prompt).toContain("fill");
      expect(prompt).toContain("select");
      expect(prompt).toContain("hover");
      expect(prompt).toContain("scroll");
      expect(prompt).toContain("keyboard");
      expect(prompt).toContain("wait");
      expect(prompt).toContain("done");
      expect(prompt).toContain("stuck");
    });

    it("documents action-specific parameters", () => {
      const prompt = buildDecisionPrompt(createPerception(), createGoal(), []);

      expect(prompt).toContain('"selector":');
      expect(prompt).toContain('"valueKey":');
      expect(prompt).toContain('"direction":');
      expect(prompt).toContain('"pixels":');
      expect(prompt).toContain('"key":');
      expect(prompt).toContain('"ms":');
      expect(prompt).toContain('"reason":');
    });
  });
});
