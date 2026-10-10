import { fireEvent, render, screen } from '@testing-library/react';
import { createRoutesStub } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import { loadAllScansFixture } from '../../test/load-fixture';
import { ReportView, type ReportViewProps } from './ReportView';

function renderView(overrides: Partial<ReportViewProps> = {}) {
	const Stub = createRoutesStub([
		{
			path: '/',
			Component: () => (
				<ReportView
					jobId="job"
					report={null}
					screenshots={[]}
					status="complete"
					error={null}
					toolbar={null}
					comparisonPanel={null}
					artifactsPanel={null}
					{...overrides}
				/>
			)
		}
	]);
	return render(<Stub />);
}

describe('ReportView acquisition states', () => {
	it('announces unavailable reports and exposes report retry', () => {
		const retry = vi.fn();
		renderView({ reportError: 'Report fetch failed: 500', onRetryReport: retry });
		expect(screen.getByRole('alert').textContent).toContain('Report unavailable');
		expect(screen.queryByText('Preparing report…')).toBeNull();
		fireEvent.click(screen.getByRole('button', { name: 'Retry loading report' }));
		expect(retry).toHaveBeenCalledOnce();
	});
	it('keeps preparing while the report is not ready', () => {
		renderView();
		expect(screen.getByRole('heading', { name: 'Preparing report…' })).toBeTruthy();
		expect(screen.queryByRole('alert')).toBeNull();
	});
	it.each(['failed', 'cancelled'] as const)('preserves the %s job state', (status) => {
		renderView({ status, error: 'Job stopped', reportError: 'Report fetch failed' });
		expect(
			screen.getByRole('heading', { name: status === 'failed' ? 'Scan failed' : 'Scan cancelled' })
		).toBeTruthy();
		expect(screen.queryByText('Report unavailable')).toBeNull();
	});
	it('shows an archived report even when live retrieval failed', () => {
		renderView({ report: loadAllScansFixture(), reportError: 'Live report unavailable' });
		expect(screen.queryByText('Report unavailable')).toBeNull();
		expect(screen.getAllByRole('tab').length).toBeGreaterThan(0);
	});
});
