import { defineConfig } from 'vitest/config';

export default defineConfig({
	test: {
		environment: 'jsdom',
		// Globals let @testing-library/react auto-clean the DOM between tests.
		globals: true,
		include: ['app/**/*.test.{ts,tsx}']
	}
});
