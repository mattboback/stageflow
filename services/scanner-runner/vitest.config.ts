import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    include: ["tests/**/*.{test,spec}.{ts,tsx}"],
    exclude: ["dist/**", "node_modules/**"],
    passWithNoTests: false,
    coverage: {
      provider: "v8",
      reporter: ["text", "lcov", "json-summary"],
      reportsDirectory: "./coverage",
      include: ["src/**/*.ts"],
      exclude: [
        "dist/**",
        "node_modules/**",
        "src/ai/**",
        "src/screenshots/**",
        "src/types/**",
        "src/index.ts",
        "src/lib.ts",
        "src/worker.ts",
      ],
      thresholds: {
        statements: 60,
        branches: 55,
        functions: 60,
        lines: 60,
      },
    },
  },
});
