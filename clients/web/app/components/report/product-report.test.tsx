import { fireEvent, render, screen, within } from '@testing-library/react';
import { createRoutesStub } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import { loadAllScansFixture } from '../../test/load-fixture';
import { buildOccurrenceModeReport } from '../../lib/report';
import type { IssueDetail, UnifiedReport } from '../../lib/types/unified-report';
import type { ProjectDiff } from '../../lib/projects';
import { AllClearBanner } from './AllClearBanner';
import { BaselineDiffSummary } from './BaselineDiffSummary';
import { FixTheseFirst } from './FixTheseFirst';
import { IssueDetailModal } from './IssueDetailModal';
import { IssueEvidenceSection } from './IssueEvidenceSection';
import { IssueRowCard } from './IssueRowCard';
import { LighthouseDetails } from './LighthouseDetails';
import { ReportHeader } from './ReportHeader';
import { ReportView } from './ReportView';
import { TruncatedEvidence } from './TruncatedEvidence';

const fixture = loadAllScansFixture();
function finding(overrides: Partial<IssueDetail> = {}): IssueDetail {
	return {
		id: 'finding',
		scanner: 'axe',
		ruleId: 'image-alt',
		severity: 'serious',
		title: 'Missing alternative text',
		description: 'Images need alternatives',
		pageId: 'page-v2-home',
		pageUrl: 'https://example.com/',
		elementCount: 1,
		occurrences: [{ selector: 'img', html: '<img src="photo.jpg">', elementId: 'occ-v2-first' }],
		...overrides
	};
}
function report(issues: IssueDetail[]): UnifiedReport {
	return {
		...fixture,
		issues,
		scanners: fixture.scanners.map((scanner) => ({ ...scanner, status: 'success' })),
		pages: [{ ...fixture.pages[0]!, id: 'page-v2-home', url: 'https://example.com/', path: '/' }],
		summary: { ...fixture.summary, totalIssues: issues.length }
	};
}
function diff(issue: IssueDetail): ProjectDiff {
	return {
		schema: 'stageflow/diff@v1',
		baseline: { jobId: 'base', totalIssues: 1, score: 100 },
		current: { jobId: 'current', totalIssues: 2, score: 90 },
		delta: { newIssues: 1, fixedIssues: 0, unchangedIssues: 1, scoreDelta: -10 },
		new: [issue],
		fixed: []
	};
}

describe('product report priorities', () => {
	it('excludes human checks, ranks severity then affected pages, caps at five and opens the representative', () => {
		const selected = vi.fn();
		const manual = finding({
			id: 'human',
			scanner: 'lighthouse',
			ruleId: 'manual',
			title: 'Verify controls manually',
			severity: 'info',
			occurrences: [],
			elementCount: 0
		});
		const wide = [
			finding({ id: 'wide-one', ruleId: 'wide', title: 'Wide problem' }),
			finding({ id: 'wide-two', ruleId: 'wide', title: 'Wide problem', pageId: 'page-v2-about' })
		];
		const critical = finding({
			id: 'critical',
			ruleId: 'critical',
			title: 'Critical problem',
			severity: 'critical'
		});
		render(
			<FixTheseFirst
				issues={[
					manual,
					...wide,
					critical,
					...Array.from({ length: 5 }, (_, index) =>
						finding({
							id: `minor-${index}`,
							ruleId: `minor-${index}`,
							title: `Minor ${index}`,
							severity: 'minor'
						})
					)
				]}
				onIssueSelect={selected}
			/>
		);
		const actions = screen.getAllByRole('button');
		expect(actions).toHaveLength(5);
		expect(actions[0]?.textContent).toContain('Critical problem');
		expect(actions[1]?.textContent).toContain('Wide problem');
		expect(actions[1]?.textContent).toContain('2 pages');
		expect(screen.queryByText('Verify controls manually')).toBeNull();
		fireEvent.click(actions[0]!);
		expect(selected).toHaveBeenCalledWith(critical);
	});
	it('hides site score and all-clear for incomplete page coverage even if an old report contains a score', () => {
		const partial = { ...report([]), scanners: [{ id: 'axe', status: 'partial' as const }] };
		const { container } = render(
			<>
				<ReportHeader report={partial} />
				<AllClearBanner report={partial} />
			</>
		);
		expect(screen.getByText('Partial')).toBeTruthy();
		expect(screen.queryByText(/Site score/)).toBeNull();
		expect(screen.queryByText('All clear.')).toBeNull();
		expect(container.textContent).not.toContain('nothing to fix');
	});
	it('does not invent element counts for node-less findings', () => {
		const pageIssue = finding({
			scanner: 'lighthouse',
			ruleId: 'link-text',
			occurrences: [],
			elementCount: 1,
			elementsTruncated: true,
			scannerData: {
				details: {
					type: 'table',
					headings: [{ key: 'text', label: 'Link Text', valueType: 'text' }],
					items: [{ text: 'Learn more' }]
				}
			}
		});
		const { container } = render(
			<>
				<IssueRowCard
					issue={finding({ scanner: 'security-headers', occurrences: [], elementCount: 1 })}
				/>
				<IssueRowCard issue={pageIssue} />
				<IssueEvidenceSection issue={pageIssue} page={null} pageOverviewUrl={null} />
			</>
		);
		expect(container.querySelector('.irow__count')).toBeNull();
		expect(screen.getByRole('cell', { name: 'Learn more' })).toBeTruthy();
		expect(screen.queryByText(/more elements without captured HTML or crops/)).toBeNull();
	});
	it('renders scanner guidance and curated impact in the default fix dialog', () => {
		render(
			<IssueDetailModal
				issue={finding({
					howToFix: 'Add the exact header: Example: safe',
					userImpact: { statement: 'People using screen readers need meaningful alternatives.' }
				})}
				page={null}
				issues={[]}
				screenshots={[]}
				onClose={vi.fn()}
			/>
		);
		expect(screen.getByText('Add the exact header: Example: safe')).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Who this affects' })).toBeTruthy();
		expect(screen.getByText(/People using screen readers/)).toBeTruthy();
	});
});

describe('Lighthouse detail evidence', () => {
	it('escapes scanner HTML and bounds table rows and cell sizes', () => {
		const { container } = render(
			<LighthouseDetails
				details={{
					type: 'table',
					headings: [
						{ key: 'node', label: 'Element' },
						{ key: 'value', label: 'Value' }
					],
					items: Array.from({ length: 40 }, () => ({
						node: { snippet: '<script>alert(1)</script>' },
						value: 'x'.repeat(700)
					}))
				}}
			/>
		);
		expect(screen.getAllByRole('row')).toHaveLength(31);
		expect(container.querySelector('script')).toBeNull();
		expect(screen.getByText('Showing 30 of 40 evidence rows.')).toBeTruthy();
		expect(screen.getAllByRole('cell')[1]?.textContent).toHaveLength(500);
	});
	it('handles malformed or unsupported detail shapes without rendering raw objects', () => {
		const { container } = render(
			<LighthouseDetails details={{ type: 'table', headings: [null, {}], items: [null, 'bad'] }} />
		);
		expect(container.textContent).toBe('');
	});
});

describe('regression presentation', () => {
	it('opens the precise new occurrence from a reusable comparison summary', () => {
		const open = vi.fn();
		const change = finding({ occurrences: [{ elementId: 'occ-v2-new' }] });
		render(<BaselineDiffSummary diff={diff(change)} onOpenIssue={open} />);
		expect(
			screen.getByRole('heading', { name: 'New in this deploy (1) · Fixed (0)' })
		).toBeTruthy();
		fireEvent.click(screen.getByRole('button', { name: /Missing alternative text/ }));
		expect(open).toHaveBeenCalledWith('occ-v2-new');
	});
	it('filters NEW findings by occurrence identity and opens the correct occurrence', () => {
		const source = finding({
			elementCount: 2,
			occurrences: [
				{ selector: '#old', html: '<img id="old">', elementId: 'occ-v2-old' },
				{ selector: '#new', html: '<img id="new">', elementId: 'occ-v2-new' }
			]
		});
		const change = { ...source, occurrences: [source.occurrences![1]!], elementCount: 1 };
		const Stub = createRoutesStub([
			{
				path: '/',
				Component: () => (
					<ReportView
						jobId="current"
						report={report([source])}
						screenshots={[]}
						status="complete"
						error={null}
						banner={null}
						artifactsPanel={null}
						comparison={diff(change)}
					/>
				)
			}
		]);
		render(<Stub initialEntries={['/?section=issues&group=flat&new=1']} />);
		const panel = screen.getByRole('tabpanel');
		expect(within(panel).getAllByText('NEW')).toHaveLength(1);
		expect(panel.querySelectorAll('.irow')).toHaveLength(1);
		fireEvent.click(panel.querySelector('button.irow')!);
		fireEvent.click(screen.getByRole('tab', { name: 'Evidence' }));
		expect(screen.getByText('#new')).toBeTruthy();
		expect(screen.queryByText('#old')).toBeNull();
	});
	it('retains all occurrence IDs and reports capped rich evidence independently', () => {
		const source = finding({
			elementCount: 7,
			elementsTruncated: true,
			occurrences: Array.from({ length: 7 }, (_, index) => ({
				selector: `#node-${index}`,
				elementId: `occ-v2-${index}`,
				...(index < 5 ? { html: '<img>' } : {})
			}))
		});
		const expanded = buildOccurrenceModeReport(report([source]));
		expect(expanded.issues).toHaveLength(7);
		expect(expanded.issues.map((issue) => issue.occurrences?.[0]?.elementId)).toEqual(
			source.occurrences!.map((occ) => occ.elementId)
		);
		render(<TruncatedEvidence issue={expanded.issues[0]!} />);
		expect(screen.getByText(/\+2 more elements/)).toBeTruthy();
	});
});
