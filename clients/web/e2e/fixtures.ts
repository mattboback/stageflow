import { expect, test as base } from '@playwright/test';

export { expect };

export const test = base.extend<{ browserErrors: void }>({
	browserErrors: [
		async ({ context, page }, use) => {
			const errors: string[] = [];
			const monitor = (candidate: typeof page) => {
				candidate.on('pageerror', (error) => errors.push(`pageerror: ${String(error)}`));
				candidate.on('console', (message) => {
					if (message.type() === 'error') errors.push(`console: ${message.text()}`);
				});
			};
			monitor(page);
			context.on('page', monitor);
			await use();
			expect(errors, 'Unexpected browser errors').toEqual([]);
		},
		{ auto: true }
	]
});
