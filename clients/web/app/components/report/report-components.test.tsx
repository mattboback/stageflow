import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { createRoutesStub } from 'react-router';
import { describe, expect, it, vi } from 'vitest';

import { loadAllScansFixture } from '../../test/load-fixture';
import { explainScore, scannerLabel } from '../../lib/report';
import type { ProjectDiff } from '../../lib/projects';
import type { UnifiedReport } from '../../lib/types/unified-report';

import { AllClearBanner } from './AllClearBanner';
import { IssueRowCard } from './IssueRowCard';
import { LighthouseSummary } from './LighthouseSummary';
import { ReportHeader } from './ReportHeader';
import { ScannerText } from './ScannerText';
import { SeverityBar, SeverityLegend } from './SeverityBar';
import { VisualReviewPanel } from './VisualReviewPanel';
import { ManualReviewTab } from './verify/ManualReviewTab';
import { VerifyContrastTab } from './verify/VerifyContrastTab';

const report = loadAllScansFixture();
const zeroSeverity: UnifiedReport['summary']['bySeverity'] = {
	critical: 0,
	serious: 0,
	moderate: 0,
	minor: 0
};

describe('ReportHeader', () => {
	it('renders the title and scan totals from the fixture', () => {
		render(<ReportHeader report={report} />);

		expect(screen.getByRole('heading', { level: 1 })).toBeTruthy();

		const totals = screen.getByRole('group', { name: 'Scan totals' });
		expect(totals.textContent).toContain(
			`${report.summary.totalIssues.toLocaleString()} occurrences`
		);
		expect(totals.textContent).toContain(`${report.summary.pagesScanned} pages scanned`);
		expect(totals.textContent).not.toContain('Human review');
	});

	it('reads "nothing to fix" instead of severity counts on a clean report', () => {
		const clean: UnifiedReport = {
			...report,
			issues: [],
			summary: { ...report.summary, totalIssues: 0, pagesScanned: 1, bySeverity: zeroSeverity }
		};
		render(<ReportHeader report={clean} />);

		const totals = screen.getByLabelText('Scan totals');
		// The page count stays: with one page there is no Pages tab to give it.
		expect(totals.textContent).toContain('1 page scanned · nothing to fix');
		expect(totals.textContent).not.toContain('critical');
	});

	it('turns severity counts and scanner chips into findings filters', () => {
		const onReviewSeverity = vi.fn();
		const onSelectScanner = vi.fn();
		render(
			<ReportHeader
				report={report}
				onReviewSeverity={onReviewSeverity}
				onSelectScanner={onSelectScanner}
			/>
		);

		const critical = report.summary.bySeverity.critical ?? 0;
		expect(critical).toBeGreaterThan(0);
		fireEvent.click(screen.getByRole('button', { name: `${critical} critical` }));
		expect(onReviewSeverity).toHaveBeenCalledWith('critical');

		const scanners = screen.getByRole('group', { name: 'Scanners' });
		for (const scanner of report.scanners) {
			expect(within(scanners).getByText(scannerLabel(scanner.id, scanner.name))).toBeTruthy();
		}
		const first = report.scanners[0]!;
		fireEvent.click(
			within(scanners).getByText(scannerLabel(first.id, first.name)).closest('button')!
		);
		expect(onSelectScanner).toHaveBeenCalledWith(first.id);
	});

	it('marks the baseline on the dial only when the comparison describes the shown score', () => {
		const shown = report.summary.score!;
		const comparison = (current: number): ProjectDiff => ({
			schema: 'stageflow/diff@v1',
			baseline: { jobId: 'base', totalIssues: 1, score: shown + 3 },
			current: { jobId: 'current', totalIssues: 2, score: current },
			delta: { newIssues: 1, fixedIssues: 0, unchangedIssues: 1, scoreDelta: current - shown - 3 },
			new: [],
			fixed: []
		});

		const view = render(<ReportHeader report={report} comparison={comparison(shown)} />);
		expect(
			screen.getByRole('img', {
				name: new RegExp(`Site score ${shown} of 100, down 3 from baseline ${shown + 3}$`)
			})
		).toBeTruthy();
		expect(view.container.querySelector('.rhead__delta')?.textContent).toBe(`▼ 3 vs ${shown + 3}`);

		view.rerender(<ReportHeader report={report} comparison={comparison(shown - 1)} />);
		expect(
			screen.getByRole('img', { name: new RegExp(`Site score ${shown} of 100$`) })
		).toBeTruthy();
		expect(view.container.querySelector('.rhead__delta')).toBeNull();
	});

	it('breaks the score down only when the breakdown reproduces it', () => {
		const demo = JSON.parse(
			readFileSync(path.resolve('public/demo/report.json'), 'utf8')
		) as UnifiedReport;
		const view = render(<ReportHeader report={demo} scoreModel={explainScore(demo)} />);
		fireEvent.click(screen.getByText('How is this scored?'));
		const table = screen.getByRole('table', { name: 'What’s lowering the score' });
		const rows = within(table)
			.getAllByRole('row')
			.slice(1)
			.map((row) => row.textContent);
		expect(rows[0]).toBe(`${scannerLabel('axe')}44%67`);
		expect(rows).toHaveLength(5);

		// The contract fixture's recorded score does not follow from its findings.
		view.rerender(<ReportHeader report={report} scoreModel={explainScore(report)} />);
		expect(screen.getByText(/starts at 100/)).toBeTruthy();
		expect(screen.queryByRole('table')).toBeNull();
	});

	it('says expected staging observations are left out only when the score leaves them out', () => {
		const demo = JSON.parse(
			readFileSync(path.resolve('public/demo/report.json'), 'utf8')
		) as UnifiedReport;
		const view = render(<ReportHeader report={demo} />);
		expect(screen.getByText(/Expected staging observations and findings/)).toBeTruthy();

		// An older report scores from its summary, which counts them.
		const legacy: UnifiedReport = { ...demo };
		delete legacy.assessment;
		view.rerender(<ReportHeader report={legacy} />);
		expect(screen.queryByText(/Expected staging observations/)).toBeNull();
		expect(screen.getByText(/Findings that need a human check are left out/)).toBeTruthy();
	});
});

describe('VisualReviewPanel', () => {
	it('selects the occurrence that is on the screenshot when its numbered group is chosen', () => {
		// Revealing a finding asks whether to animate; jsdom has no matchMedia.
		vi.stubGlobal('matchMedia', (query: string) => ({
			matches: false,
			media: query,
			addEventListener: () => {},
			removeEventListener: () => {}
		}));
		try {
			const page = report.pages[0]!;
			const base = {
				scanner: 'axe',
				ruleId: 'image-alt',
				severity: 'serious' as const,
				title: 'Images must have alternative text',
				description: '',
				pageId: page.id,
				pageUrl: page.url,
				elementCount: 1
			};
			// The first occurrence has no marker, as when its element was off-screen.
			const located: UnifiedReport = {
				...report,
				issues: [
					{ ...base, id: 'unlocated' },
					{ ...base, id: 'located' }
				],
				pages: [
					{
						...page,
						pageOverview: {
							screenshotFilename: 'page.png',
							pageWidth: 1280,
							pageHeight: 2400,
							elements: [
								{
									issueId: 'located',
									ruleId: 'image-alt',
									severity: 'serious',
									selector: 'img',
									nodeIndex: 0,
									xPercent: 10,
									yPercent: 10,
									widthPercent: 10,
									heightPercent: 5,
									x: 128,
									y: 240,
									width: 128,
									height: 120
								}
							]
						}
					}
				]
			};
			const onIssueSelect = vi.fn();
			render(
				<VisualReviewPanel
					report={located}
					screenshots={[
						{
							kind: 'page_overview',
							artifact_id: 'shot',
							scanner_id: 'axe',
							page_id: page.id,
							url: '/shot.webp'
						}
					]}
					activeScanner={null}
					activePage={page.id}
					getReviewVerdict={() => null}
					onSelectPage={vi.fn()}
					onIssueSelect={onIssueSelect}
				/>
			);
			const zoom = screen.getByRole<HTMLButtonElement>('button', { name: 'Zoom to finding' });
			expect(zoom.disabled).toBe(true);

			fireEvent.click(screen.getByRole('button', { name: /Images must have alternative text/ }));
			expect(onIssueSelect).toHaveBeenCalledWith(expect.objectContaining({ id: 'located' }));
			expect(zoom.disabled).toBe(false);
		} finally {
			vi.unstubAllGlobals();
		}
	});
});

describe('SeverityBar', () => {
	const counts = [
		{ severity: 'critical' as const, count: 1 },
		{ severity: 'minor' as const, count: 3 }
	];

	it('is an image only when it is named, and scales against a shared total', () => {
		const view = render(<SeverityBar counts={counts} scale={8} />);
		const bar = view.container.querySelector('.sevbar')!;
		expect(bar.getAttribute('aria-hidden')).toBe('true');
		const grows = [...bar.children].map((segment) => (segment as HTMLElement).style.flexGrow);
		expect(grows).toEqual(['0.125', '0.375', '0.5']);

		view.rerender(<SeverityBar counts={counts} label="4 findings: 1 critical and 3 minor" />);
		expect(screen.getByRole('img', { name: '4 findings: 1 critical and 3 minor' })).toBeTruthy();
		expect(view.container.querySelector('.sevbar__rest')).toBeNull();
	});

	it('names each count in text, as buttons when it filters', () => {
		const select = vi.fn();
		render(<SeverityLegend counts={counts} onSelect={select} />);
		fireEvent.click(screen.getByRole('button', { name: '3 minor' }));
		expect(select).toHaveBeenCalledWith('minor');
	});
});

describe('ReportHeader on a partial report', () => {
	const unscoredSummary = { ...report.summary };
	delete unscoredSummary.score;
	delete unscoredSummary.scoreGrade;
	const partial: UnifiedReport = {
		...report,
		summary: unscoredSummary,
		scanners: [
			...report.scanners.filter((scanner) => scanner.id !== 'axe'),
			{ id: 'axe', status: 'failed', error: 'axe ran out of memory (limit 3072 MB)' }
		]
	};

	it('says which scanner did not finish and shows no score', () => {
		render(<ReportHeader report={partial} />);

		expect(screen.getByText('Partial')).toBeTruthy();
		expect(screen.queryByText('Completed')).toBeNull();
		expect(screen.getByRole('status').textContent).toContain('did not finish');
		expect(screen.queryByText(/Site score/)).toBeNull();
	});

	it('reads Completed when every scanner finished', () => {
		const complete: UnifiedReport = {
			...report,
			scanners: report.scanners.map((scanner) => ({ ...scanner, status: 'success' as const }))
		};
		render(<ReportHeader report={complete} />);

		expect(screen.getByText('Completed')).toBeTruthy();
		expect(screen.queryByText('Partial')).toBeNull();
	});
});

describe('AllClearBanner', () => {
	function cleanReport(errors: UnifiedReport['errors'] = []): UnifiedReport {
		return {
			...report,
			issues: [],
			errors,
			summary: { ...report.summary, totalIssues: 0, pagesWithIssues: 0, bySeverity: zeroSeverity }
		};
	}

	function renderBanner(clean: UnifiedReport) {
		const Stub = createRoutesStub([
			{ path: '/', Component: () => <AllClearBanner report={clean} /> }
		]);
		return render(<Stub initialEntries={['/']} />);
	}

	it('celebrates a clean run and points at saving a local baseline', () => {
		renderBanner(cleanReport());

		expect(screen.getByRole('heading', { name: 'All clear.' })).toBeTruthy();
		expect(screen.getByText('0 findings')).toBeTruthy();
		expect(screen.getByRole('link', { name: /Save this as a local baseline/ })).toBeTruthy();
		expect(screen.queryByText(/check the Artifacts tab/)).toBeNull();
	});

	it('flags report errors before recommending the run as a baseline', () => {
		renderBanner(
			cleanReport([
				{
					scope: 'scanner',
					scannerId: 'axe',
					code: 'timeout',
					message: 'timed out',
					retryable: true
				}
			])
		);

		expect(screen.getByText(/One report error was noted/)).toBeTruthy();
	});
});

describe('IssueRowCard', () => {
	const issue = report.issues[0]!;

	it('renders title, severity badge, scanner, and occurrence count', () => {
		render(<IssueRowCard issue={issue} />);

		expect(screen.getByRole('heading', { level: 3 }).textContent).toBeTruthy();
		expect(screen.getByText(issue.severity)).toBeTruthy();
		if (issue.ruleId) {
			expect(screen.getByText(issue.ruleId)).toBeTruthy();
		}
	});

	it('invokes onSelect with the issue when clicked', () => {
		const onSelect = vi.fn();
		render(<IssueRowCard issue={issue} onSelect={onSelect} />);

		fireEvent.click(screen.getByRole('button'));
		expect(onSelect).toHaveBeenCalledWith(issue);
	});
});

describe('LighthouseSummary', () => {
	it('shows the 0–1 contract score as a 0–100 Lighthouse score', () => {
		render(
			<LighthouseSummary
				categories={[{ id: 'accessibility', title: 'Accessibility', avgScore: 0.92 }]}
			/>
		);
		expect(screen.getByText('92')).toBeTruthy();
	});
});

describe('ScannerText', () => {
	it('renders markdown code spans and links from scanner prose', () => {
		render(
			<p>
				<ScannerText text="Use a `<meta>` tag. [Learn more](https://example.com/docs)." />
			</p>
		);
		expect(screen.getByText('<meta>').tagName).toBe('CODE');
		const link = screen.getByRole('link', { name: 'Learn more' });
		expect(link.getAttribute('href')).toBe('https://example.com/docs');
	});

	it('drops link URLs but keeps the text when links are disabled', () => {
		render(
			<p data-testid="no-links">
				<ScannerText text="[Learn more](https://example.com/docs) about charsets." links={false} />
			</p>
		);
		const el = screen.getByTestId('no-links');
		expect(el.textContent).toBe('Learn more about charsets.');
		expect(el.querySelector('a')).toBeNull();
	});
});

describe('ManualReviewTab', () => {
	it('records and clears a generic human-review decision', () => {
		const issue = {
			...report.issues[0]!,
			id: 'manual-review-component-test',
			scanner: 'lighthouse',
			ruleId: 'custom-controls-labels',
			severity: 'info' as const,
			description: 'Check custom controls for accessible labels.',
			occurrences: []
		};

		render(<ManualReviewTab issue={issue} jobId="manual-review-component-job" />);
		fireEvent.click(screen.getByRole('button', { name: 'Mark pass' }));

		expect(screen.getByText('Reviewed · pass')).toBeTruthy();
		fireEvent.click(screen.getByRole('button', { name: 'Clear verdict' }));
		expect(screen.getByRole('button', { name: 'Mark pass' })).toBeTruthy();
	});
});

describe('VerifyContrastTab', () => {
	it('records and displays the text classification and threshold used', () => {
		const issue = {
			...report.issues[0]!,
			id: 'contrast-review-component-test',
			scanner: 'axe',
			ruleId: 'color-contrast',
			description: 'Verify rendered contrast.',
			scannerData: {
				axeIncomplete: true,
				contrastData: {
					fgColor: '#898989',
					bgColor: '#ffffff',
					fontSize: '16px',
					fontWeight: '400'
				}
			}
		};

		render(
			<VerifyContrastTab
				issue={issue}
				page={null}
				pageOverviewUrl={null}
				jobId="contrast-review-component-job"
			/>
		);
		fireEvent.click(screen.getByRole('checkbox', { name: /Large text/ }));
		fireEvent.click(screen.getByRole('button', { name: 'Mark pass' }));

		expect(screen.getByText('Verified · pass')).toBeTruthy();
		expect(screen.getByText(/large text/).textContent).toContain('3.0:1 required');
	});
});
