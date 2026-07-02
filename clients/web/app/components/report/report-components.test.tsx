import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { loadAllScansFixture } from '../../test/load-fixture';

import { IssueRowCard } from './IssueRowCard';
import { OverviewDashboard } from './OverviewDashboard';
import { ReportHeader } from './ReportHeader';
import { SeverityBreakdown } from './SeverityBreakdown';

const report = loadAllScansFixture();

describe('ReportHeader', () => {
	it('renders the score, page/scanner counts, and severity totals from the fixture', () => {
		render(<ReportHeader report={report} />);

		expect(screen.getByRole('heading', { name: 'Scan report' })).toBeTruthy();
		expect(
			screen.getByText(
				`${report.summary.pagesScanned} pages · ${report.scanners.length} scanners`
			)
		).toBeTruthy();

		const totals = screen.getByLabelText('Issue counts');
		expect(totals.textContent).toContain(`Total${report.summary.totalIssues}`);
		expect(totals.textContent).toContain(`Critical${report.summary.bySeverity.critical ?? 0}`);
	});
});

describe('SeverityBreakdown', () => {
	it('renders one row per severity level with fixture counts', () => {
		render(<SeverityBreakdown bySeverity={report.summary.bySeverity} />);

		const list = screen.getByRole('list', { name: 'Issues by severity' });
		const rows = list.querySelectorAll('li');
		expect(rows.length).toBe(5);
		expect(screen.getByText('Critical').parentElement?.textContent).toContain('Critical');
	});

	it('renders zero counts when no severity data is present', () => {
		render(<SeverityBreakdown bySeverity={undefined} />);
		const list = screen.getByRole('list', { name: 'Issues by severity' });
		expect(list.querySelectorAll('li').length).toBe(5);
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

describe('OverviewDashboard', () => {
	const handlers = () => ({
		onSelectPage: vi.fn(),
		onSelectScanner: vi.fn(),
		onSearchIssues: vi.fn(),
		onReviewSeverity: vi.fn()
	});

	it('summarizes the fixture: issue patterns, scanner statuses, top pages', () => {
		render(<OverviewDashboard report={report} {...handlers()} />);

		expect(screen.getByRole('heading', { name: /issue patterns? found/ })).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Scanner Status' })).toBeTruthy();
		for (const scanner of report.scanners) {
			expect(screen.getAllByText(scanner.name ?? scanner.id).length).toBeGreaterThan(0);
		}
	});

	it('routes the urgent CTA to the worst outstanding severity', () => {
		const h = handlers();
		render(<OverviewDashboard report={report} {...h} />);

		const critical = report.summary.bySeverity.critical ?? 0;
		expect(critical).toBeGreaterThan(0);
		fireEvent.click(
			screen.getByRole('button', { name: `Fix ${critical} critical first →` })
		);
		expect(h.onReviewSeverity).toHaveBeenCalledWith('critical');
	});
});
