import type { IssueDetail, PageSummary } from '$lib/types/unified-report';

import IssueEvidenceSection from '$lib/components/report/IssueEvidenceSection.svelte';
import { cleanup, render } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

function createPage(overrides?: Partial<PageSummary>): PageSummary {
	return {
		id: 'page-1',
		url: 'http://example.com',
		path: '/home',
		issueCount: 2,
		durationMs: 1000,
		pageOverview: {
			screenshotFilename: 'overview.png',
			pageWidth: 1200,
			pageHeight: 800,
			elements: [
				{
					issueId: 'issue-1',
					ruleId: 'color-contrast',
					severity: 'critical',
					selector: '.hero-text',
					nodeIndex: 0,
					xPercent: 10,
					yPercent: 20,
					widthPercent: 30,
					heightPercent: 10,
					x: 120,
					y: 160,
					width: 360,
					height: 80
				},
				{
					issueId: 'issue-1',
					ruleId: 'color-contrast',
					severity: 'critical',
					selector: '.subtitle',
					nodeIndex: 1,
					xPercent: 10,
					yPercent: 35,
					widthPercent: 25,
					heightPercent: 5,
					x: 120,
					y: 280,
					width: 300,
					height: 40
				}
			]
		},
		...overrides
	};
}

function createIssue(overrides?: Partial<IssueDetail>): IssueDetail {
	return {
		id: 'issue-1',
		scanner: 'axe',
		ruleId: 'color-contrast',
		severity: 'critical',
		title: 'Low contrast text',
		description: 'Text elements have insufficient contrast ratio',
		pageId: 'page-1',
		pageUrl: 'http://example.com',
		elementCount: 2,
		occurrences: [
			{
				selector: '.hero-text',
				html: '<h1 class="hero-text">Welcome</h1>',
				failureSummary: 'Increase contrast ratio to at least 4.5:1'
			},
			{
				selector: '.subtitle',
				html: '<p class="subtitle">Subtitle text</p>',
				failureSummary: 'Increase contrast ratio to at least 4.5:1'
			}
		],
		...overrides
	};
}

/** The full-page overview SVG (with interactive rects) is always last and
 * carries an aria-label mentioning highlighted elements. */
function getOverviewSvg(container: HTMLElement): SVGElement | null {
	return container.querySelector('svg[aria-label*="highlighted"]');
}

describe('IssueEvidenceSection', () => {
	afterEach(() => {
		cleanup();
	});

	describe('scanner screenshot rendering', () => {
		it('renders scanner-captured fallback img when no overview crop is available', () => {
			const { getByAltText } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					// No pageOverview elements -> no crop SVG, falls through to <img>
					page: createPage({
						pageOverview: {
							screenshotFilename: 'overview.png',
							pageWidth: 1200,
							pageHeight: 800,
							elements: []
						}
					}),
					screenshotUrl: 'http://example.com/screenshot.png',
					pageOverviewUrl: null,
					onElementClick: vi.fn()
				}
			});

			expect(getByAltText('Issue highlighted on page')).toBeInTheDocument();
		});

		it('does not render scanner screenshot fallback when screenshotUrl is null and no crop possible', () => {
			const { queryByAltText } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue({ id: 'unmatched-issue' }),
					page: createPage(),
					screenshotUrl: null,
					pageOverviewUrl: null,
					onElementClick: vi.fn()
				}
			});

			expect(queryByAltText('Issue highlighted on page')).not.toBeInTheDocument();
		});
	});

	describe('SVG-based page overview rendering', () => {
		it('renders the full-page overview SVG with viewBox matching page dimensions', () => {
			const page = createPage();
			const { container } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page,
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			const svg = getOverviewSvg(container);
			expect(svg).toBeInTheDocument();
			expect(svg?.getAttribute('viewBox')).toBe('0 0 1200 800');
			expect(svg?.getAttribute('preserveAspectRatio')).toBe('xMinYMin meet');
		});

		it('renders SVG image element with correct href and dimensions', () => {
			const page = createPage();
			const { container } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page,
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			const overviewSvg = getOverviewSvg(container);
			const image = overviewSvg?.querySelector('image');
			expect(image).toBeInTheDocument();
			expect(image?.getAttribute('href')).toBe('http://example.com/overview.png');
			expect(image?.getAttribute('width')).toBe('1200');
			expect(image?.getAttribute('height')).toBe('800');
		});

		it('renders rect elements at exact pixel coordinates for issue elements', () => {
			const page = createPage();
			const { container } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page,
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			const overviewSvg = getOverviewSvg(container);
			const rects = overviewSvg?.querySelectorAll('rect') ?? [];
			expect(rects).toHaveLength(2);

			expect(rects[0].getAttribute('x')).toBe('120');
			expect(rects[0].getAttribute('y')).toBe('160');
			expect(rects[0].getAttribute('width')).toBe('360');
			expect(rects[0].getAttribute('height')).toBe('80');

			expect(rects[1].getAttribute('x')).toBe('120');
			expect(rects[1].getAttribute('y')).toBe('280');
			expect(rects[1].getAttribute('width')).toBe('300');
			expect(rects[1].getAttribute('height')).toBe('40');
		});

		it('only renders elements matching the current issue', () => {
			const page = createPage({
				pageOverview: {
					screenshotFilename: 'overview.png',
					pageWidth: 1200,
					pageHeight: 800,
					elements: [
						{
							issueId: 'issue-1',
							ruleId: 'color-contrast',
							severity: 'critical',
							selector: '.hero-text',
							nodeIndex: 0,
							xPercent: 10,
							yPercent: 20,
							widthPercent: 30,
							heightPercent: 10,
							x: 120,
							y: 160,
							width: 360,
							height: 80
						},
						{
							issueId: 'other-issue',
							ruleId: 'image-alt',
							severity: 'serious',
							selector: '.logo',
							nodeIndex: 0,
							xPercent: 5,
							yPercent: 5,
							widthPercent: 10,
							heightPercent: 5,
							x: 60,
							y: 40,
							width: 120,
							height: 40
						}
					]
				}
			});

			const { container } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue({ id: 'issue-1' }),
					page,
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			const overviewSvg = getOverviewSvg(container);
			const rects = overviewSvg?.querySelectorAll('rect') ?? [];
			expect(rects).toHaveLength(1);
			expect(rects[0].getAttribute('x')).toBe('120');
		});
	});

	describe('element click interaction', () => {
		it('calls onElementClick with correct elementId when overview rect is clicked', async () => {
			const user = userEvent.setup();
			const onElementClick = vi.fn();

			const { container } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page: createPage(),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick
				}
			});

			const overviewSvg = getOverviewSvg(container);
			const rect = overviewSvg?.querySelector('rect');
			expect(rect).toBeTruthy();

			await user.click(rect as Element);

			expect(onElementClick).toHaveBeenCalledWith('issue-1-el-0');
		});

		it('calls onElementClick when Enter key is pressed on overview rect', () => {
			const onElementClick = vi.fn();

			const { container } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page: createPage(),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick
				}
			});

			const overviewSvg = getOverviewSvg(container);
			const rect = overviewSvg?.querySelector('rect');
			expect(rect).toBeTruthy();

			rect?.dispatchEvent(
				new KeyboardEvent('keydown', {
					key: 'Enter',
					bubbles: true,
					cancelable: true
				})
			);

			expect(onElementClick).toHaveBeenCalledWith('issue-1-el-0');
		});

		it('shows helper text when there are clickable elements', () => {
			const { getByText } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page: createPage(),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			expect(getByText(/click a highlight box to jump/i)).toBeInTheDocument();
		});
	});

	describe('edge cases', () => {
		it('does not render full-page overview when showPageOverview is false', () => {
			const { container, queryByText } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page: createPage(),
					screenshotUrl: 'http://example.com/screenshot.png',
					pageOverviewUrl: 'http://example.com/overview.png',
					showPageOverview: false,
					onElementClick: vi.fn()
				}
			});

			expect(queryByText('On the page')).not.toBeInTheDocument();
			expect(getOverviewSvg(container)).toBeNull();
		});

		it('does not render full-page overview when page is null', () => {
			const { container, queryByText } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page: null,
					screenshotUrl: 'http://example.com/screenshot.png',
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			expect(queryByText('On the page')).not.toBeInTheDocument();
			expect(getOverviewSvg(container)).toBeNull();
		});

		it('does not render full-page overview when pageOverviewUrl is null', () => {
			const { container, queryByText } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page: createPage(),
					screenshotUrl: 'http://example.com/screenshot.png',
					pageOverviewUrl: null,
					onElementClick: vi.fn()
				}
			});

			expect(queryByText('On the page')).not.toBeInTheDocument();
			expect(getOverviewSvg(container)).toBeNull();
		});

		it('does not render full-page overview when page dimensions are invalid', () => {
			const { container, queryByText } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page: createPage({
						pageOverview: {
							screenshotFilename: 'overview.png',
							pageWidth: 0,
							pageHeight: 800,
							elements: []
						}
					}),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			expect(queryByText('On the page')).not.toBeInTheDocument();
			expect(getOverviewSvg(container)).toBeNull();
		});

		it('does not show helper text when there are no matching elements', () => {
			const { queryByText } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue({ id: 'non-existent-issue' }),
					page: createPage(),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			expect(queryByText(/click a highlight box/i)).not.toBeInTheDocument();
		});
	});

	describe('page-level evidence fallback', () => {
		it('renders the full-page screenshot for a page-global issue with no element match', () => {
			const { container, getByText, queryByText } = render(IssueEvidenceSection, {
				props: {
					// Issue id has no matching pageOverview element (page-global finding).
					issue: createIssue({ id: 'missing-csp', occurrences: [] }),
					page: createPage({
						pageOverview: {
							screenshotFilename: 'overview.png',
							pageWidth: 1200,
							pageHeight: 800,
							elements: []
						}
					}),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			const fallbackSvg = container.querySelector('svg[aria-label*="page-level finding"]');
			expect(fallbackSvg).toBeInTheDocument();
			expect(fallbackSvg?.querySelector('image')?.getAttribute('href')).toBe(
				'http://example.com/overview.png'
			);
			// No bounding-box rects, and never the "no evidence" empty state.
			expect(fallbackSvg?.querySelector('rect')).toBeNull();
			expect(getByText(/applies to the whole page/i)).toBeInTheDocument();
			expect(queryByText(/no dom evidence captured/i)).not.toBeInTheDocument();
		});

		it('does not show the page-level fallback for manual review items', () => {
			const { container, getByText } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue({ scanner: 'lighthouse', severity: 'info', occurrences: [] }),
					page: createPage({
						pageOverview: {
							screenshotFilename: 'overview.png',
							pageWidth: 1200,
							pageHeight: 800,
							elements: []
						}
					}),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			expect(container.querySelector('svg[aria-label*="page-level finding"]')).toBeNull();
			expect(getByText(/manual review item/i)).toBeInTheDocument();
		});
	});

	describe('manual review', () => {
		it('renders a manual review banner for Lighthouse info issues without occurrences', () => {
			const { getByText } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue({
						scanner: 'lighthouse',
						severity: 'info',
						occurrences: []
					}),
					page: createPage(),
					screenshotUrl: null,
					pageOverviewUrl: null,
					onElementClick: vi.fn()
				}
			});

			expect(getByText(/manual review item/i)).toBeInTheDocument();
		});
	});

	describe('accessibility', () => {
		it('has appropriate ARIA label on the full-page overview SVG', () => {
			const { container } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page: createPage(),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			const svg = getOverviewSvg(container);
			expect(svg?.getAttribute('role')).toBe('img');
			expect(svg?.getAttribute('aria-label')).toContain('2 highlighted element');
		});

		it('has appropriate ARIA labels on interactive overview rects', () => {
			const { container } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page: createPage(),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			const overviewSvg = getOverviewSvg(container);
			const rect = overviewSvg?.querySelector('rect');
			expect(rect?.getAttribute('role')).toBe('button');
			expect(rect?.getAttribute('tabindex')).toBe('0');
			expect(rect?.getAttribute('aria-label')).toContain('Highlight occurrence');
		});

		it('has title elements for tooltips on overview rects', () => {
			const { container } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue(),
					page: createPage(),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			const overviewSvg = getOverviewSvg(container);
			const titles = overviewSvg?.querySelectorAll('rect title') ?? [];
			expect(titles).toHaveLength(2);
			expect(titles[0].textContent).toContain('Click to focus occurrence 1');
			expect(titles[1].textContent).toContain('Click to focus occurrence 2');
		});
	});

	describe('severity stroke color', () => {
		it('applies correct stroke color based on issue severity in overlay rects', () => {
			const { container } = render(IssueEvidenceSection, {
				props: {
					issue: createIssue({ severity: 'critical' }),
					page: createPage(),
					screenshotUrl: null,
					pageOverviewUrl: 'http://example.com/overview.png',
					onElementClick: vi.fn()
				}
			});

			const overviewSvg = getOverviewSvg(container);
			const rect = overviewSvg?.querySelector('rect');
			const strokeColor = rect?.getAttribute('stroke');
			expect(strokeColor).toContain('rgba(239, 68, 68');
		});
	});
});
