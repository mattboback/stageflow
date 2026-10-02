import { render, screen, within } from '@testing-library/react';
import { createRoutesStub } from 'react-router';
import { describe, expect, it, vi } from 'vitest';
import { useScanStatus } from '../lib/hooks/useScanMonitor';
import type { ScanResult } from '../lib/types/scan';
import Scan from './scan';
import { scannerLabel } from '../lib/report';

vi.mock('../lib/hooks/useScanMonitor', () => ({ useScanStatus: vi.fn() }));

function renderScanning(scannerStatuses?: ScanResult['scanner_statuses']) {
	vi.mocked(useScanStatus).mockReturnValue({
		status: 'scanning',
		result: {
			id: 'test-scan',
			state: 'SCANNING',
			created_at: '2026-10-02T00:00:00Z',
			updated_at: '2026-10-02T00:00:00Z',
			expected_scanners: ['axe', 'lighthouse'],
			remaining_scanners: ['axe', 'lighthouse'],
			...(scannerStatuses ? { scanner_statuses: scannerStatuses } : {})
		},
		elapsed: 5,
		logs: [],
		transport: 'streaming',
		error: null,
		retry: vi.fn()
	});
	const Routes = createRoutesStub([{ path: '/scan/:id', Component: Scan }]);
	render(<Routes initialEntries={['/scan/test-scan']} />);
}

function channel(name: string) {
	const row = screen.getByText(name).closest('.ch');
	if (!(row instanceof HTMLElement)) throw new Error(`Missing scanner row: ${name}`);
	return within(row);
}

describe('scanner launch status', () => {
	it('shows a durably pending scanner as waiting while another scanner runs', () => {
		renderScanning({
			axe: { state: 'running', pages_scanned: 1, total_pages: 3, violations: 0 },
			lighthouse: { state: 'pending', pages_scanned: 0, total_pages: 3, violations: 0 }
		});
		expect(channel('Lighthouse').getByText('Waiting')).toBeTruthy();
		expect(channel(scannerLabel('axe')).getByText('Running')).toBeTruthy();
		expect(channel(scannerLabel('axe')).getByText('Page progress: 1 of 3')).toBeTruthy();
	});

	it('retains the running fallback for older snapshots without scanner states', () => {
		renderScanning();
		expect(channel('Lighthouse').getByText('Running')).toBeTruthy();
	});
});

describe('queued scan', () => {
	function renderQueued(queuePosition: number) {
		vi.mocked(useScanStatus).mockReturnValue({
			status: 'pending',
			result: {
				id: 'test-scan',
				state: 'PENDING',
				queue_position: queuePosition,
				created_at: '2026-10-02T00:00:00Z',
				updated_at: '2026-10-02T00:00:00Z',
				expected_scanners: ['axe', 'lighthouse'],
				remaining_scanners: ['axe', 'lighthouse']
			},
			elapsed: 5,
			logs: [],
			transport: 'streaming',
			error: null,
			retry: vi.fn()
		});
		const Routes = createRoutesStub([{ path: '/scan/:id', Component: Scan }]);
		render(<Routes initialEntries={['/scan/test-scan']} />);
	}

	it('says how many scans start first', () => {
		renderQueued(2);
		expect(screen.getByText('Queued, 2 ahead')).toBeTruthy();
		expect(screen.getByRole('heading', { name: 'Waiting in the queue' })).toBeTruthy();
		expect(screen.getByText('Queued')).toBeTruthy();
	});

	it('says when the scan is next', () => {
		renderQueued(0);
		expect(screen.getByText('Queued, next in line')).toBeTruthy();
	});
});
