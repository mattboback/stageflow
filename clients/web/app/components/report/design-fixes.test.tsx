import { fireEvent, render, screen } from '@testing-library/react';
import { createRoutesStub } from 'react-router';
import { describe, expect, it, vi } from 'vitest';

import type { IssueDetail, IssueGroup, PageSummary } from '../../lib/types/unified-report';
import { SiteHeader } from '../SiteHeader';
import { ElementCrop } from './ElementCrop';
import { IssueDetailModal } from './IssueDetailModal';
import { IssueGroupRow } from './IssueGroupRow';

function finding(overrides: Partial<IssueDetail> = {}): IssueDetail {
	return {
		id: 'finding',
		scanner: 'axe',
		ruleId: 'image-alt',
		severity: 'serious',
		title: 'Missing alternative text',
		description: 'Images need alternatives',
		pageId: 'page-1',
		pageUrl: 'https://example.com/technology',
		elementCount: 1,
		occurrences: [{ selector: 'img.hero', html: '<img class="hero">', elementId: 'occ-1' }],
		...overrides
	};
}

describe('design fixes', () => {
	it('keeps primary navigation on app (scan/report) routes', () => {
		const Stub = createRoutesStub([
			{
				path: '/',
				Component: () => (
					<SiteHeader app={{ backTo: '/playground', backLabel: 'New scan', section: 'Report' }} />
				)
			}
		]);
		render(<Stub />);
		expect(screen.getByRole('link', { name: 'Projects' })).toBeTruthy();
		expect(screen.getByRole('link', { name: 'Configure a scan' })).toBeTruthy();
		expect(screen.getByRole('link', { name: /New scan/ })).toBeTruthy();
	});

	it('distinguishes identical-page occurrence rows by selector and number', () => {
		const page: PageSummary = {
			id: 'page-1',
			url: 'https://example.com/technology',
			path: '/technology',
			issueCount: 2,
			durationMs: 1
		};
		const group: IssueGroup = {
			fingerprint: 'g1',
			ruleId: 'image-alt',
			scanner: 'axe',
			title: 'Missing alternative text',
			description: 'd',
			severity: 'serious',
			occurrences: [
				finding({ id: 'o1' }),
				finding({
					id: 'o2',
					occurrences: [
						{ selector: 'div.dropdown-toggle-2', html: '<div>x</div>', elementId: 'occ-2' }
					]
				})
			],
			pageIds: ['page-1']
		};
		render(<IssueGroupRow group={group} pageById={new Map([[page.id, page]])} defaultOpen />);
		expect(screen.getByText('Occurrence #1')).toBeTruthy();
		expect(screen.getByText('Occurrence #2')).toBeTruthy();
		expect(screen.getByText('div.dropdown-toggle-2')).toBeTruthy();
	});

	it('badges a rule on every page as site-wide', () => {
		const page: PageSummary = {
			id: 'page-1',
			url: 'https://example.com/technology',
			path: '/technology',
			issueCount: 1,
			durationMs: 1
		};
		const page2: PageSummary = {
			id: 'page-2',
			url: 'https://example.com/about',
			path: '/about',
			issueCount: 1,
			durationMs: 1
		};
		const group: IssueGroup = {
			fingerprint: 'g-wide',
			ruleId: 'html-has-lang',
			scanner: 'axe',
			title: 'Html missing lang',
			description: 'd',
			severity: 'serious',
			occurrences: [finding({ id: 'w1' }), finding({ id: 'w2', pageId: 'page-2' })],
			pageIds: ['page-1', 'page-2']
		};
		render(
			<IssueGroupRow
				group={group}
				pageById={
					new Map([
						[page.id, page],
						[page2.id, page2]
					])
				}
				totalPages={2}
				defaultOpen
			/>
		);
		expect(screen.getByText('Site-wide (all 2 pages)')).toBeTruthy();
	});

	it('shows a top-viewport crop with badge for document-level elements', () => {
		const page: PageSummary = {
			id: 'page-1',
			url: 'https://example.com/',
			path: '/',
			issueCount: 1,
			durationMs: 1,
			pageOverview: {
				screenshotFilename: 'shot.png',
				pageWidth: 1280,
				pageHeight: 4000,
				elements: []
			}
		};
		const { container } = render(
			<ElementCrop
				page={page}
				overviewUrl="https://example.com/shot.png"
				element={{
					issueId: 'f1',
					ruleId: 'landmark-one-main',
					selector: 'html',
					nodeIndex: 0,
					x: 0,
					y: 0,
					width: 1280,
					height: 4000,
					severity: 'serious',
					xPercent: 0,
					yPercent: 0,
					widthPercent: 100,
					heightPercent: 100
				}}
				ariaLabel="Document crop"
			/>
		);
		expect(screen.getByText(/Document-level finding/)).toBeTruthy();
		expect(container.querySelector('svg')?.getAttribute('viewBox')).toBe('0 0 1280 720');
	});

	it('shows rule, WCAG and affected-group metadata at the top of Fix', () => {
		render(
			<IssueDetailModal
				issue={finding({
					wcagTags: ['wcag2a', 'wcag131'],
					helpUrl: 'https://dequeuniversity.com/rules/axe/image-alt',
					userImpact: {
						statement: 'Screen reader users miss the image.',
						affectedGroups: ['blind'],
						userStory: ''
					}
				})}
				page={null}
				issues={[finding()]}
				screenshots={[]}
				onClose={vi.fn()}
			/>
		);
		expect(screen.getByText('Rule & scanner')).toBeTruthy();
		expect(screen.getAllByText(/axe/).length).toBeGreaterThan(0);
		expect(screen.getAllByText(/wcag131/).length).toBeGreaterThan(0);
		expect(screen.getAllByText('Blind users').length).toBeGreaterThan(0);
	});

	it('hides Verify this fix without a Verify tab and keeps Save note in the modal footer', () => {
		const { container } = render(
			<IssueDetailModal
				issue={finding()}
				page={null}
				issues={[finding()]}
				screenshots={[]}
				onClose={vi.fn()}
				onVerify={vi.fn()}
			/>
		);
		expect(screen.queryByRole('button', { name: 'Verify this fix' })).toBeNull();
		const foot = container.querySelector('.imodal__foot');
		expect(foot).toBeTruthy();
		expect(foot?.querySelector('button')?.textContent ?? foot?.textContent).toBeTruthy();
		expect(screen.getByRole('button', { name: 'Save note' })).toBeTruthy();
		expect(foot?.contains(screen.getByRole('button', { name: 'Save note' }))).toBe(true);
	});

	it('shares a mobile content region with notes and focuses the close action', () => {
		const issue = finding();
		const { container } = render(
			<IssueDetailModal
				issue={issue}
				page={null}
				issues={[issue]}
				screenshots={[]}
				onClose={vi.fn()}
			/>
		);
		const content = container.querySelector('.imodal__content');
		expect(content?.contains(container.querySelector('.imodal__body'))).toBe(true);
		expect(content?.contains(screen.getByRole('button', { name: 'Save note' }))).toBe(true);
		expect(document.activeElement).toBe(screen.getByRole('button', { name: 'Close modal' }));
	});

	it('resets both responsive scroll regions when the finding or tab changes', () => {
		const issue = finding();
		const props = { page: null, issues: [issue], screenshots: [], onClose: vi.fn() };
		const { container, rerender } = render(<IssueDetailModal {...props} issue={issue} />);
		const regions = container.querySelectorAll<HTMLElement>('.imodal__content, .imodal__body');
		regions.forEach((region) => {
			region.scrollTop = 300;
		});
		fireEvent.click(screen.getByRole('tab', { name: 'Evidence' }));
		regions.forEach((region) => {
			expect(region.scrollTop).toBe(0);
		});
		regions.forEach((region) => {
			region.scrollTop = 300;
		});
		rerender(<IssueDetailModal {...props} issue={finding({ id: 'next-finding' })} />);
		regions.forEach((region) => {
			expect(region.scrollTop).toBe(0);
		});
	});
});
