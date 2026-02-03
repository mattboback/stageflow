#!/usr/bin/env node

const path = require("node:path");
const { createRequire } = require("node:module");

const requireFromRunner = createRequire(
  path.resolve(__dirname, "../../platform/scanner-runner/package.json")
);

let chromium;
let AxeBuilder;
try {
  ({ chromium } = requireFromRunner("playwright"));
  AxeBuilder = requireFromRunner("@axe-core/playwright").default;
} catch (err) {
  console.error(
    "[test-axe-local] Missing scanner-runner deps. Run: (cd platform/scanner-runner && bun install --frozen-lockfile)"
  );
  console.error(err);
  process.exit(1);
}

async function test() {
  const url = process.argv[2] ?? "https://matthewboback.com";

  let browser;
  try {
    browser = await chromium.launch({ headless: true });
    const context = await browser.newContext();
    const page = await context.newPage();

    console.log(`Navigating to ${url}...`);
    await page.goto(url, { waitUntil: "networkidle" });

    console.log("Waiting 500ms for dynamic content...");
    await page.waitForTimeout(500);

    console.log("Running axe-core analysis...");
    const axe = new AxeBuilder({ page });
    const results = await axe.analyze();

    console.log("\n=== AXE RESULTS ===");
    console.log("Violations:", results.violations.length);
    console.log("Passes:", results.passes.length);
    console.log("Incomplete:", results.incomplete.length);
    console.log("Inapplicable:", results.inapplicable.length);

    if (results.violations.length > 0) {
      console.log("\n=== VIOLATIONS ===");
      for (const violation of results.violations) {
        console.log(`\n${violation.id} (${violation.impact})`);
        console.log(`  Help: ${violation.help}`);
        console.log(`  Nodes: ${violation.nodes.length}`);
        if (violation.nodes.length > 0) {
          console.log(
            `  First node: ${violation.nodes[0].html?.substring(0, 100)}`
          );
        }
      }
    }
  } finally {
    if (browser) {
      try {
        await browser.close();
      } catch (err) {
        console.warn("[test-axe-local] Failed to close browser:", err);
      }
    }
  }
}

test().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
