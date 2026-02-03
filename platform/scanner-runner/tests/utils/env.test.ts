import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { getEnvBool, getEnvInt, getEnvNumber, getEnvString } from "../../src/utils/env";

const ORIGINAL_ENV = process.env;

function resetEnv(): void {
  process.env = { ...ORIGINAL_ENV };
}

describe("env utils", () => {
  beforeEach(() => {
    resetEnv();
  });

  afterEach(() => {
    resetEnv();
  });

  it("parses boolean values with defaults", () => {
    process.env.FLAG = "true";
    expect(getEnvBool("FLAG", false)).toBe(true);

    process.env.FLAG = "0";
    expect(getEnvBool("FLAG", true)).toBe(false);

    delete process.env.FLAG;
    expect(getEnvBool("FLAG", true)).toBe(true);
  });

  it("parses numbers and ints safely", () => {
    process.env.VALUE = "3.14";
    expect(getEnvNumber("VALUE", 1)).toBe(3.14);
    expect(getEnvInt("VALUE", 1)).toBe(3);

    process.env.VALUE = "nope";
    expect(getEnvNumber("VALUE", 2)).toBe(2);
    expect(getEnvInt("VALUE", 2)).toBe(2);
  });

  it("returns trimmed strings with default fallback", () => {
    process.env.NAME = "  stageflow ";
    expect(getEnvString("NAME", "default")).toBe("stageflow");

    delete process.env.NAME;
    expect(getEnvString("NAME", "default")).toBe("default");
  });
});
