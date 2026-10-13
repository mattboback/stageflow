import { render, screen } from '@testing-library/react';
import { createRoutesStub } from 'react-router';
import { expect, it, vi } from 'vitest';
import { loadAllScansFixture } from '../../test/load-fixture';
import { LocalBaselineComparison } from './LocalBaselineComparison';

it('keeps partial reports reviewable but prevents accepting them as a baseline', () => {
	const report = loadAllScansFixture();
	report.scanners[0]!.status = 'failed';
	const Stub = createRoutesStub([
		{
			path: '/',
			Component: () => (
				<LocalBaselineComparison
					project={{
						id: 'site',
						name: 'Site',
						schemaVersion: 1,
						createdAt: 'today',
						updatedAt: 'today',
						configuration: {
							urls: ['https://example.com'],
							scanners: [],
							browser: 'chromium',
							highlightStyle: 'solid'
						}
					}}
					run={{
						projectId: 'site',
						jobId: report.meta.jobId,
						configFingerprint: 'settings',
						status: 'complete',
						createdAt: 'today',
						report
					}}
					baseline={null}
					report={report}
					promoting={false}
					message={null}
					onPromote={vi.fn()}
					onOpenIssue={vi.fn()}
				/>
			)
		}
	]);
	render(<Stub initialEntries={['/']} />);
	expect(
		(screen.getByRole('button', { name: 'Promote as baseline' }) as HTMLButtonElement).disabled
	).toBe(true);
	expect(screen.getByRole('status').textContent).toContain('incomplete scanner coverage');
	expect(screen.getByRole('link', { name: /Run again/ })).toBeDefined();
});
