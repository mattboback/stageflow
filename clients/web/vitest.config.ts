import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [sveltekit()],
	resolve: {
		conditions: ['browser', 'svelte', 'development']
	},
	test: {
		include: ['tests/**/*.{test,spec}.ts'],
		exclude: ['node_modules/**', '.svelte-kit/**'],
		environment: 'jsdom',
		setupFiles: ['tests/setup.ts'],
		coverage: {
			provider: 'v8',
			reporter: ['text', 'lcov', 'json-summary'],
			include: [
				'src/lib/api/utils.ts',
				'src/lib/components/nav/MobileMenu.svelte',
				'src/lib/components/report/IssueEvidenceSection.svelte',
				'src/lib/components/report/PageOverviewViewer.svelte',
				'src/lib/components/report/SeverityBreakdown.svelte',
				'src/lib/components/scan-status/ProcessingView.svelte',
				'src/lib/components/scan-status/ScanStatusContent.svelte',
				'src/lib/components/ui/PageSection.svelte',
				'src/lib/components/ui/Progress.svelte',
				'src/lib/components/ui/SelectField.svelte',
				'src/lib/components/ui/TerminalCardHeader.svelte',
				'src/lib/report/filters.ts',
				'src/lib/report/occurrence-mode.ts',
				'src/lib/report/scanner-summary.ts',
				'src/lib/report/screenshots.ts',
				'src/lib/report/severity.ts',
				'src/lib/report/virtualization.ts',
				'src/lib/stores/scan-status/constants.ts',
				'src/lib/stores/scan-status/log-messages.ts',
				'src/lib/stores/scan-status/scanner-progress.ts',
				'src/lib/utils/**/*.ts'
			],
			exclude: ['**/*.stories.ts', '**/story-harnesses/**', '**/index.ts'],
			thresholds: {
				statements: 85,
				branches: 80,
				functions: 90,
				lines: 85
			}
		}
	}
});
