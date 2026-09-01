import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createRoutesStub } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { loadAllScansFixture } from '../../test/load-fixture';
import { deleteScanJob } from '../../lib/api/client';

import { ReportJobActions } from './ReportJobActions';

vi.mock('../../lib/api/client', () => ({
	deleteScanJob: vi.fn()
}));

const report = loadAllScansFixture();
const deleteScanJobMock = vi.mocked(deleteScanJob);

afterEach(() => {
	vi.unstubAllGlobals();
	deleteScanJobMock.mockReset();
});

function renderActions(
	props: Partial<Parameters<typeof ReportJobActions>[0]> = {},
	initialPath = '/scan/job-actions-1/report'
) {
	const Stub = createRoutesStub([
		{
			path: '/scan/:id/report',
			Component: () => (
				<ReportJobActions
					jobId="job-actions-1"
					report={report}
					archived={false}
					canDelete
					showSave
					onSaveProject={vi.fn(async () => undefined)}
					{...props}
				/>
			)
		},
		{ path: '/', Component: () => <p>home</p> }
	]);

	return render(<Stub initialEntries={[initialPath]} />);
}

describe('ReportJobActions', () => {
	it('copies the current report URL', async () => {
		const writeText = vi.fn(async () => undefined);
		vi.stubGlobal('navigator', { clipboard: { writeText } });
		renderActions();

		fireEvent.click(screen.getByRole('button', { name: 'Copy share link' }));

		await waitFor(() => {
			expect(writeText).toHaveBeenCalled();
			expect(screen.getByRole('button', { name: 'Link copied' })).toBeTruthy();
		});
	});

	it('shows a save error without leaving the report', async () => {
		const onSaveProject = vi.fn(async () => {
			throw new Error('IndexedDB is unavailable');
		});
		renderActions({ onSaveProject });

		fireEvent.click(screen.getByRole('button', { name: 'Save in this browser' }));

		expect(await screen.findByRole('alert')).toHaveTextContent('IndexedDB is unavailable');
		expect(onSaveProject).toHaveBeenCalled();
	});

	it('confirms delete and navigates home after a successful erase request', async () => {
		deleteScanJobMock.mockResolvedValue(undefined);
		renderActions();

		fireEvent.click(screen.getByRole('button', { name: 'Delete this scan' }));
		expect(screen.getByRole('heading', { name: 'Delete this scan?' })).toBeTruthy();
		expect(screen.getByText(/durable job record is not erased/)).toBeTruthy();

		fireEvent.click(screen.getByRole('button', { name: 'Delete scan' }));

		expect(await screen.findByText('home')).toBeTruthy();
		expect(deleteScanJobMock).toHaveBeenCalledWith('job-actions-1');
	});

	it('keeps the report visible when delete is rejected because the job is still running', async () => {
		deleteScanJobMock.mockRejectedValue(new Error('This scan is still running.'));
		renderActions();

		fireEvent.click(screen.getByRole('button', { name: 'Delete this scan' }));
		fireEvent.click(screen.getByRole('button', { name: 'Delete scan' }));

		expect(await screen.findByRole('alert')).toHaveTextContent('This scan is still running.');
		expect(screen.getByRole('button', { name: 'Delete this scan' })).toBeTruthy();
	});

	it('hides delete on archived local copies', () => {
		renderActions({ archived: true, canDelete: false, showSave: false });

		expect(screen.queryByRole('button', { name: 'Delete this scan' })).toBeNull();
		expect(screen.getByText(/stored in this browser/i)).toBeTruthy();
	});
});
