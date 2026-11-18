import { defineConfig } from '@playwright/test';

// E2E smoke against the production build with Nginx-equivalent prerender and
// __spa-fallback.html routing. StageFlow API calls are mocked in each spec.
export default defineConfig({
	testDir: 'e2e',
	timeout: 30_000,
	// Give each browser the runner's resources in CI; local runs stay parallel.
	// https://playwright.dev/docs/ci#workers
	...(process.env.CI ? { workers: 1 } : {}),
	use: {
		baseURL: 'http://127.0.0.1:4173',
		trace: 'retain-on-failure'
	},
	webServer: {
		command: 'bun run build && node qa/serve-build.mjs',
		url: 'http://127.0.0.1:4173',
		reuseExistingServer: !process.env.CI,
		timeout: 60_000
	}
});
