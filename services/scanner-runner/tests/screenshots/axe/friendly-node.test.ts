/**
 * Friendly Node Tests
 */

import type { Locator, Page } from 'playwright';

import { describe, expect, it, vi } from 'vitest';

import { buildFriendlyNodeInfo } from '../../../src/screenshots/axe/friendly-node';

const createMockLocator = (evaluateResult: unknown, shouldThrow = false): Locator => {
	const mockLocator = {
		first: vi.fn().mockReturnThis(),
		evaluate: shouldThrow
			? vi.fn().mockRejectedValue(new Error('Element not found'))
			: vi.fn().mockResolvedValue(evaluateResult)
	} as unknown as Locator;
	return mockLocator;
};

const createMockPage = (locatorResult: Locator): Page => {
	return {
		locator: vi.fn().mockReturnValue(locatorResult)
	} as unknown as Page;
};

describe('buildFriendlyNodeInfo', () => {
	describe('label generation', () => {
		it('builds label for a button element', async () => {
			const mockLocator = createMockLocator({
				tagName: 'button',
				role: 'button',
				name: 'Submit',
				region: 'Main content',
				textSnippet: 'Submit'
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'button.submit');

			expect(result).toBeDefined();
			expect(result?.label).toContain('Button');
			expect(result?.label).toContain('Submit');
			expect(result?.tagName).toBe('button');
		});

		it('builds label for a link element', async () => {
			const mockLocator = createMockLocator({
				tagName: 'a',
				role: 'link',
				name: 'Learn more',
				region: 'Navigation',
				textSnippet: 'Learn more about our services'
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'a.cta');

			expect(result?.label).toContain('Link');
			expect(result?.label).toContain('Learn more');
			expect(result?.tagName).toBe('a');
		});

		it('builds label for an image element', async () => {
			const mockLocator = createMockLocator({
				tagName: 'img',
				role: 'img',
				name: 'Company logo',
				region: 'Header',
				textSnippet: undefined
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'img.logo');

			expect(result?.label).toContain('Image');
			expect(result?.label).toContain('Company logo');
		});

		it('includes region in label', async () => {
			const mockLocator = createMockLocator({
				tagName: 'div',
				role: undefined,
				name: undefined,
				region: 'Footer',
				textSnippet: 'Copyright 2024'
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'div.copyright');

			expect(result?.label).toContain('in Footer');
			expect(result?.region).toBe('Footer');
		});

		it('handles element without name', async () => {
			const mockLocator = createMockLocator({
				tagName: 'div',
				role: undefined,
				name: undefined,
				region: 'Main content',
				textSnippet: 'Some content'
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'div.container');

			expect(result?.label).toBeDefined();
			expect(result?.tagName).toBe('div');
		});
	});

	describe('role handling', () => {
		it('uses role for label when present', async () => {
			const mockLocator = createMockLocator({
				tagName: 'div',
				role: 'navigation',
				name: 'Main menu',
				region: 'Header',
				textSnippet: undefined
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, "div[role='navigation']");

			expect(result?.role).toBe('navigation');
			expect(result?.label).toBeDefined();
		});

		it('handles button role on non-button element', async () => {
			const mockLocator = createMockLocator({
				tagName: 'div',
				role: 'button',
				name: 'Click me',
				region: 'Page body',
				textSnippet: 'Click me'
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, "div[role='button']");

			expect(result?.label).toContain('Button');
		});

		it('handles link role on non-anchor element', async () => {
			const mockLocator = createMockLocator({
				tagName: 'span',
				role: 'link',
				name: 'Go somewhere',
				region: 'Main content',
				textSnippet: 'Go somewhere'
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, "span[role='link']");

			expect(result?.label).toContain('Link');
		});
	});

	describe('name sources', () => {
		it('uses aria-label when available', async () => {
			const mockLocator = createMockLocator({
				tagName: 'button',
				role: 'button',
				name: 'Close dialog',
				region: 'Dialog',
				textSnippet: ''
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'button.close');

			expect(result?.name).toBe('Close dialog');
		});

		it('uses alt text for images', async () => {
			const mockLocator = createMockLocator({
				tagName: 'img',
				role: undefined,
				name: 'Product thumbnail',
				region: 'Main content',
				textSnippet: undefined
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'img.product');

			expect(result?.name).toBe('Product thumbnail');
		});
	});

	describe('region detection', () => {
		it('identifies main landmark', async () => {
			const mockLocator = createMockLocator({
				tagName: 'button',
				role: 'button',
				name: 'Submit',
				region: 'Main content',
				textSnippet: 'Submit'
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'main button');

			expect(result?.region).toBe('Main content');
		});

		it('identifies header landmark', async () => {
			const mockLocator = createMockLocator({
				tagName: 'nav',
				role: 'navigation',
				name: 'Primary',
				region: 'Header',
				textSnippet: undefined
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'header nav');

			expect(result?.region).toBe('Header');
		});

		it('identifies footer landmark', async () => {
			const mockLocator = createMockLocator({
				tagName: 'a',
				role: 'link',
				name: 'Privacy Policy',
				region: 'Footer',
				textSnippet: 'Privacy Policy'
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'footer a');

			expect(result?.region).toBe('Footer');
		});
	});

	describe('type labels mapping', () => {
		it('maps input to Input field label', async () => {
			const mockLocator = createMockLocator({
				tagName: 'input',
				role: undefined,
				name: undefined,
				region: 'Main content',
				textSnippet: undefined
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'input');

			expect(result?.label).toContain('Input');
		});

		it('maps h1 to Heading label', async () => {
			const mockLocator = createMockLocator({
				tagName: 'h1',
				role: undefined,
				name: undefined,
				region: 'Main content',
				textSnippet: undefined
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'h1');

			expect(result?.label).toContain('Heading');
		});
	});

	describe('error handling', () => {
		it('returns undefined when locator throws', async () => {
			const mockLocator = createMockLocator(null, true);
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'invalid-selector');

			expect(result).toBeUndefined();
		});

		it('returns undefined when element is not found', async () => {
			const mockLocator = {
				first: vi.fn().mockReturnThis(),
				evaluate: vi.fn().mockRejectedValue(new Error('Element detached'))
			} as unknown as Locator;
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'removed-element');

			expect(result).toBeUndefined();
		});

		it('returns undefined when evaluate returns null', async () => {
			const mockLocator = createMockLocator(null);
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'some-selector');

			expect(result).toBeUndefined();
		});
	});

	describe('selector inclusion', () => {
		it('includes selector in the result', async () => {
			const selector = '#my-unique-button';
			const mockLocator = createMockLocator({
				tagName: 'button',
				role: 'button',
				name: 'Click',
				region: 'Main content',
				textSnippet: 'Click'
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, selector);

			expect(result?.selector).toBe(selector);
		});
	});

	describe('text snippet', () => {
		it('captures text snippet from element', async () => {
			const mockLocator = createMockLocator({
				tagName: 'p',
				role: undefined,
				name: undefined,
				region: 'Article',
				textSnippet: 'This is the beginning of a paragraph...'
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'p.intro');

			expect(result?.textSnippet).toBeDefined();
		});

		it('handles empty text content', async () => {
			const mockLocator = createMockLocator({
				tagName: 'div',
				role: undefined,
				name: undefined,
				region: 'Main content',
				textSnippet: undefined
			});
			const mockPage = createMockPage(mockLocator);

			const result = await buildFriendlyNodeInfo(mockPage, 'div.empty');

			expect(result?.textSnippet).toBeUndefined();
		});
	});
});
