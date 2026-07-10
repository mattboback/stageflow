import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { loadAllScansFixture } from '../../test/load-fixture';

import { IssueRowCard } from './IssueRowCard';
import { ReportHeader } from './ReportHeader';
import { ReportStatStrip } from './ReportStatStrip';

const report = loadAllScansFixture();

describe('ReportHeader', () => {
	it('renders the title and scan totals from the fixture', () => {
		render(<ReportHeader report={report} />);

		expect(screen.getByRole('heading', { level: 1 })).toBeTruthy();

		const totals = screen.getByLabelText('Scan totals');
		expect(totals.textContent).toContain('Total issues');
		expect(totals.textContent).toContain(report.summary.totalIssues.toLocaleString());
		expect(totals.textContent).toContain('Pages scanned');
		expect(totals.textContent).toContain(String(report.scanners.length));
	});
});

describe('ReportStatStrip', () => {
	const handlers = () => ({
		onReviewSeverity: vi.fn(),
		onSelectScanner: vi.fn(),
		onReviewManual: vi.fn()
	});

	it('renders severity pills and scanner chips from the fixture', () => {
		render(<ReportStatStrip report={report} {...handlers()} />);

		const critical = report.summary.bySeverity.critical ?? 0;
		expect(critical).toBeGreaterThan(0);
		expect(
			screen.getByRole('button', { name: `${critical} critical` })
		).toBeTruthy();
		for (const scanner of report.scanners) {
			expect(screen.getAllByText(scanner.name ?? scanner.id).length).toBeGreaterThan(0);
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
			screen.getAllByText(scanner.name ?? scanner.id)[0]!.closest('button')!
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

