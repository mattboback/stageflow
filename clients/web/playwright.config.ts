import { defineConfig } from '@playwright/test';

// E2E smoke against the production build served by `vite preview`, with the
// StageFlow API mocked from the committed report fixture — no backend needed.
export default defineConfig({
	testDir: 'e2e',
	timeout: 30_000,
	use: {
		baseURL: 'http://127.0.0.1:4173'
	},
	webServer: {
		command: 'bun run preview -- --host 127.0.0.1 --port 4173 --strictPort',
		url: 'http://127.0.0.1:4173',
		reuseExistingServer: !process.env.CI,
		timeout: 60_000
	}
});
