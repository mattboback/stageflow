import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { loadAllScansFixture } from '../../test/load-fixture';
import { scannerLabel } from '../../lib/report';

import { IssueRowCard } from './IssueRowCard';
import { ReportHeader } from './ReportHeader';
import { ReportStatStrip } from './ReportStatStrip';
import { ScannerText } from './ScannerText';
import { ManualReviewTab } from './verify/ManualReviewTab';

const report = loadAllScansFixture();

describe('ReportHeader', () => {
	it('renders the title and scan totals from the fixture', () => {
		render(<ReportHeader report={report} />);

		expect(screen.getByRole('heading', { level: 1 })).toBeTruthy();

		const totals = screen.getByLabelText('Scan totals');
		expect(totals.textContent).toContain('Findings');
		expect(totals.textContent).toContain(report.summary.totalIssues.toLocaleString());
		expect(totals.textContent).toContain('Pages scanned');
		expect(totals.textContent).toContain('Human review');
	});
});

describe('ReportStatStrip', () => {
	const handlers = () => ({
		onReviewSeverity: vi.fn(),
		onSelectScanner: vi.fn()
	});

	it('renders severity pills and scanner chips from the fixture', () => {
		render(<ReportStatStrip report={report} {...handlers()} />);

		const critical = report.summary.bySeverity.critical ?? 0;
		expect(critical).toBeGreaterThan(0);
		expect(screen.getByRole('button', { name: `${critical} critical` })).toBeTruthy();
		for (const scanner of report.scanners) {
			expect(screen.getAllByText(scannerLabel(scanner.id, scanner.name)).length).toBeGreaterThan(0);
		}
	});

	it('routes severity pills and scanner chips to the issues view', () => {
		const h = handlers();
		render(<ReportStatStrip report={report} {...h} />);

		const critical = report.summary.bySeverity.critical ?? 0;
		fireEvent.click(screen.getByRole('button', { name: `${critical} critical` }));
		expect(h.onReviewSeverity).toHaveBeenCalledWith('critical');

		const scanner = report.scanners[0]!;
		fireEvent.click(
			screen.getAllByText(scannerLabel(scanner.id, scanner.name))[0]!.closest('button')!
		);
		expect(h.onSelectScanner).toHaveBeenCalledWith(scanner.id);
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
