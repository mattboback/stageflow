import type { Page } from 'playwright';

import { describe, expect, it, vi } from 'vitest';

import type { AgentGoal } from '../../src/ai/types';

import { checkGoal } from '../../src/ai/goal-checker';

// Mock Playwright Page — returns an object satisfying the subset of the Page
// interface that checkGoal exercises. Cast once here so the production call
// sites in every test stay free of `as any`.
const createMockPage = (
	url: string,
	options: {
		visibleSelectors?: string[];
		visibleTexts?: string[];
	} = {}
): Page => {
	const { visibleSelectors = [], visibleTexts = [] } = options;

	const page = {
		url: vi.fn(() => url),
		locator: vi.fn((selector: string) => ({
			first: vi.fn(() => ({
				isVisible: vi.fn(() => Promise.resolve(visibleSelectors.includes(selector)))
			}))
		})),
		getByText: vi.fn((text: string) => ({
			first: vi.fn(() => ({
				isVisible: vi.fn(() => Promise.resolve(visibleTexts.includes(text)))
			}))
		}))
	};

	return page as unknown as Page;
};

function createLocatorRejectingPage(url: string, err: Error): Page {
	const isVisible = vi.fn().mockRejectedValue(err);
	const first = vi.fn().mockReturnValue({ isVisible });
	const locator = vi.fn().mockReturnValue({ first });

	return {
		url: vi.fn(() => url),
		locator,
		getByText: vi.fn()
	} as unknown as Page;
}

function createTextRejectingPage(url: string, err: Error): Page {
	const isVisible = vi.fn().mockRejectedValue(err);
	const first = vi.fn().mockReturnValue({ isVisible });
	const getByText = vi.fn().mockReturnValue({ first });

	return {
		url: vi.fn(() => url),
		locator: vi.fn(),
		getByText
	} as unknown as Page;
}

describe('checkGoal', () => {
	describe('with no success criteria', () => {
		it('returns not achieved with low confidence', async () => {
			const page = createMockPage('https://example.com');
			const goal: AgentGoal = {
				objective: 'Do something',
				successCriteria: []
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
			expect(result.confidence).toBe(0.2);
			expect(result.reason).toContain('No success criteria');
		});

		it('handles undefined successCriteria', async () => {
			const page = createMockPage('https://example.com');
			const goal: AgentGoal = {
				objective: 'Do something'
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
			expect(result.confidence).toBe(0.2);
		});
	});

	describe('url-contains criterion', () => {
		it('returns achieved when URL contains value', async () => {
			const page = createMockPage('https://example.com/dashboard');
			const goal: AgentGoal = {
				objective: 'Navigate to dashboard',
				successCriteria: [{ type: 'url-contains', value: 'dashboard' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(true);
			expect(result.confidence).toBe(1);
			expect(result.reason).toContain('All success criteria met');
		});

		it('returns not achieved when URL does not contain value', async () => {
			const page = createMockPage('https://example.com/login');
			const goal: AgentGoal = {
				objective: 'Navigate to dashboard',
				successCriteria: [{ type: 'url-contains', value: 'dashboard' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
			expect(result.confidence).toBe(0.6);
			expect(result.reason).toContain('url-contains');
			expect(result.reason).toContain('dashboard');
		});
	});

	describe('url-matches criterion', () => {
		it('returns achieved when URL matches regex', async () => {
			const page = createMockPage('https://example.com/user/12345');
			const goal: AgentGoal = {
				objective: 'View user profile',
				successCriteria: [{ type: 'url-matches', value: '/user/\\d+' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(true);
		});

		it('returns not achieved when URL does not match regex', async () => {
			const page = createMockPage('https://example.com/user/abc');
			const goal: AgentGoal = {
				objective: 'View user profile',
				successCriteria: [{ type: 'url-matches', value: '/user/\\d+$' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
		});

		it('handles invalid regex gracefully', async () => {
			const page = createMockPage('https://example.com');
			const goal: AgentGoal = {
				objective: 'Test invalid regex',
				successCriteria: [{ type: 'url-matches', value: '[invalid(' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
			expect(result.reason).toContain('url-matches');
		});
	});

	describe('element-visible criterion', () => {
		it('returns achieved when element is visible', async () => {
			const page = createMockPage('https://example.com', {
				visibleSelectors: ['#success-message']
			});
			const goal: AgentGoal = {
				objective: 'Submit form',
				successCriteria: [{ type: 'element-visible', value: '#success-message' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(true);
		});

		it('returns not achieved when element is not visible', async () => {
			const page = createMockPage('https://example.com', {
				visibleSelectors: []
			});
			const goal: AgentGoal = {
				objective: 'Submit form',
				successCriteria: [{ type: 'element-visible', value: '#success-message' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
			expect(result.reason).toContain('element-visible');
		});

		it('handles locator errors gracefully', async () => {
			const page = createLocatorRejectingPage(
				'https://example.com',
				new Error('Element not found')
			);
			const goal: AgentGoal = {
				objective: 'Find element',
				successCriteria: [{ type: 'element-visible', value: '#nonexistent' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
		});
	});

	describe('text-visible criterion', () => {
		it('returns achieved when text is visible', async () => {
			const page = createMockPage('https://example.com', {
				visibleTexts: ['Welcome back!']
			});
			const goal: AgentGoal = {
				objective: 'Log in successfully',
				successCriteria: [{ type: 'text-visible', value: 'Welcome back!' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(true);
		});

		it('returns not achieved when text is not visible', async () => {
			const page = createMockPage('https://example.com', {
				visibleTexts: []
			});
			const goal: AgentGoal = {
				objective: 'Log in successfully',
				successCriteria: [{ type: 'text-visible', value: 'Welcome back!' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
			expect(result.reason).toContain('text-visible');
		});

		it('handles getByText errors gracefully', async () => {
			const page = createTextRejectingPage('https://example.com', new Error('Text not found'));
			const goal: AgentGoal = {
				objective: 'Find text',
				successCriteria: [{ type: 'text-visible', value: 'Not there' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
		});
	});

	describe('multiple criteria', () => {
		it('returns achieved when all criteria are met', async () => {
			const page = createMockPage('https://example.com/dashboard', {
				visibleSelectors: ['#user-menu'],
				visibleTexts: ['Welcome']
			});
			const goal: AgentGoal = {
				objective: 'Complete login',
				successCriteria: [
					{ type: 'url-contains', value: 'dashboard' },
					{ type: 'element-visible', value: '#user-menu' },
					{ type: 'text-visible', value: 'Welcome' }
				]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(true);
			expect(result.confidence).toBe(1);
		});

		it('returns not achieved if any criterion fails', async () => {
			const page = createMockPage('https://example.com/dashboard', {
				visibleSelectors: ['#user-menu'],
				visibleTexts: [] // "Welcome" not visible
			});
			const goal: AgentGoal = {
				objective: 'Complete login',
				successCriteria: [
					{ type: 'url-contains', value: 'dashboard' },
					{ type: 'element-visible', value: '#user-menu' },
					{ type: 'text-visible', value: 'Welcome' }
				]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
			expect(result.reason).toContain('text-visible');
		});

		it('fails fast on first unmet criterion', async () => {
			const page = createMockPage('https://example.com/login', {
				visibleSelectors: ['#user-menu']
			});
			const goal: AgentGoal = {
				objective: 'Complete login',
				successCriteria: [
					{ type: 'url-contains', value: 'dashboard' }, // This fails first
					{ type: 'element-visible', value: '#user-menu' }
				]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
			expect(result.reason).toContain('url-contains');
			// Should not mention element-visible since we fail fast
			expect(result.reason).not.toContain('element-visible');
		});
	});

	describe('unknown criterion type', () => {
		it('returns not achieved for unknown criterion types', async () => {
			const page = createMockPage('https://example.com');
			const goal: AgentGoal = {
				objective: 'Test unknown',
				successCriteria: [{ type: 'custom', value: 'some-value' }]
			};

			const result = await checkGoal(page, goal);

			expect(result.achieved).toBe(false);
		});
	});
});
