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
				'src/lib/components/case-studies/**/*.ts',
				'src/lib/components/ui/**/*.ts',
				'src/lib/utils/**/*.ts'
			],
			exclude: ['**/*.svelte'],
			thresholds: {
				statements: 50,
				branches: 40,
				functions: 50,
				lines: 50
			}
		}
	}
});
