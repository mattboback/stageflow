import type { PreScanAction } from "../core/types";
import type { ActionDecision, AgentGoal } from "./types";

import { isPlainObject, parseFirstJsonObject } from "./json";

export function parseActionDecision(
  content: string,
  goal: AgentGoal,
): ActionDecision | undefined {
  const parsed = parseFirstJsonObject(content);
  if (!parsed) {
    return undefined;
  }

  const rawAction = parsed.action;
  if (!isPlainObject(rawAction)) {
    return undefined;
  }

  const action = parseAction(rawAction, goal);
  if (!action) {
    return undefined;
  }

  const reasoning = typeof parsed.reasoning === "string" ? parsed.reasoning : "";
  const confidence = typeof parsed.confidence === "number" ? parsed.confidence : 0.7;

  return { action, reasoning, confidence };
}

function parseAction(
  raw: Record<string, unknown>,
  goal: AgentGoal,
): ActionDecision["action"] | undefined {
  const type = raw.type;
  if (typeof type !== "string") {
    return undefined;
  }

  switch (type) {
    case "done":
      return { type: "done" };
    case "stuck":
      return { type: "stuck", reason: parseStuckReason(raw) };
    case "click":
    case "hover":
      return parseSelectorAction(type, raw);
    case "wait":
      return parseWaitAction(raw);
    case "keyboard":
      return parseKeyboardAction(raw);
    case "scroll":
      return parseScrollAction(raw);
    case "fill":
    case "select":
      return parseInputAction(type, raw, goal);
    default:
      return undefined;
  }
}

function parseStuckReason(raw: Record<string, unknown>): string {
  return typeof raw.reason === "string" ? raw.reason : "Unknown";
}

function parseSelectorAction(
  type: "click" | "hover",
  raw: Record<string, unknown>,
): PreScanAction | undefined {
  const selector = raw.selector;
  if (typeof selector !== "string" || !selector.trim()) {
    return undefined;
  }

  return type === "click" ? { type: "click", selector } : { type: "hover", selector };
}

function parseWaitAction(raw: Record<string, unknown>): PreScanAction | undefined {
  const ms = raw.ms;
  if (typeof ms !== "number" || !Number.isFinite(ms) || ms < 0) {
    return undefined;
  }

  return { type: "wait", ms };
}

function parseKeyboardAction(raw: Record<string, unknown>): PreScanAction | undefined {
  const key = raw.key;
  if (typeof key !== "string" || !key.trim()) {
    return undefined;
  }

  return { type: "keyboard", key };
}

function parseScrollAction(raw: Record<string, unknown>): PreScanAction {
  const direction = raw.direction;
  const pixels = raw.pixels;
  const selector = raw.selector;

  const action: PreScanAction = { type: "scroll" };

  if (typeof direction === "string" && (direction === "up" || direction === "down")) {
    action.direction = direction;
  }

  if (typeof pixels === "number" && Number.isFinite(pixels) && pixels > 0) {
    action.pixels = pixels;
  }

  if (typeof selector === "string" && selector.trim()) {
    action.selector = selector;
  }

  return action;
}

function parseInputAction(
  type: "fill" | "select",
  raw: Record<string, unknown>,
  goal: AgentGoal,
): ActionDecision["action"] {
  const selector = raw.selector;
  if (typeof selector !== "string" || !selector.trim()) {
    return { type: "stuck", reason: "Fill/select requires selector" };
  }

  const inputValues = goal.inputValues;
  if (!inputValues || Object.keys(inputValues).length === 0) {
    return {
      type: "stuck",
      reason: "Fill/select disallowed: no input values configured",
    };
  }

  const valueKey = raw.valueKey;
  if (typeof valueKey !== "string" || !valueKey.trim()) {
    return { type: "stuck", reason: "Fill/select requires valueKey" };
  }

  const resolved = inputValues[valueKey];
  if (resolved == null) {
    return { type: "stuck", reason: `Unknown valueKey: ${valueKey}` };
  }

  return type === "fill"
    ? { type: "fill", selector, value: resolved }
    : { type: "select", selector, value: resolved };
}
