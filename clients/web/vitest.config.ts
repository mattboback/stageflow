import { defineConfig } from 'vitest/config';

export default defineConfig({
	test: {
		environment: 'jsdom',
		// Globals let @testing-library/react auto-clean the DOM between tests.
		globals: true,
		include: ['app/**/*.test.{ts,tsx}'],
		coverage: {
			provider: 'v8',
			reporter: ['text', 'html', 'json-summary'],
			reportsDirectory: './coverage',
			include: ['app/**/*.{ts,tsx}'],
			exclude: [
				'app/**/*.test.{ts,tsx}',
				// Test scaffolding, not app code.
				'app/**/test-*.ts',
				// App wiring with no logic of its own. The Playwright suite exercises
				// these against a real production build; jsdom unit tests would only
				// restate the framework's behavior.
				'app/root.tsx',
				'app/routes.ts',
				'app/entry.client.tsx',
				'app/**/*.d.ts'
			]
			// Deliberately no `thresholds` yet. A number chosen before anyone has read
			// the report is a number nobody can defend — which is the objection to
			// scanner-runner's `statements: 60`. The Go side ratchets against a
			// checked-in baseline instead (devtools/scripts/go/coverage.sh); this
			// follows the same route once a baseline exists here.
		}
	}
});
