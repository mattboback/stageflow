import { cleanup, render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, expect, it, vi } from 'vitest';
import Playground from './playground';
import { VerifyFindingDialog } from '../components/report/VerifyFindingDialog';
import { loadAllScansFixture } from '../test/load-fixture';
import type { IssueDetail } from '../lib/types/unified-report';

vi.mock('../lib/scan-policy', () => ({ MAX_SCAN_PAGES: 25, ZIP_UPLOAD_ENABLED: false }));
vi.mock('../lib/api/client', async (original) => ({
	...(await original<typeof import('../lib/api/client')>()),
	fetchScanners: vi.fn().mockResolvedValue({ scanners: [], categories: [] })
}));
afterEach(cleanup);

it('hides ZIP upload from hosted configuration', () => {
	render(
		<MemoryRouter>
			<Playground />
		</MemoryRouter>
	);
	expect(screen.queryByRole('button', { name: 'ZIP upload' })).toBeNull();
	// With URL the only target type, there is no toggle to show.
	expect(screen.queryByRole('group', { name: 'Target type' })).toBeNull();
	expect(screen.queryByLabelText(/ZIP archive/i)).toBeNull();
	expect(screen.getByText('Add up to 25 public URLs to scan.')).toBeTruthy();
	expect(
		screen.getByRole('button', {
			name: 'Find pages on this site to scan',
			description: 'Then pick up to 25 to scan.'
		})
	).toBeTruthy();
});

it('offers public URL guidance instead of a ZIP picker when verifying archived ZIP evidence', () => {
	const report = loadAllScansFixture();
	report.meta.scanContext = { ...report.meta.scanContext!, inputType: 'zip' };
	const issue: IssueDetail = {
		id: 'issue',
		scanner: 'axe',
		ruleId: 'image-alt',
		title: 'Missing alt',
		description: 'Add alt text',
		severity: 'serious',
		pageId: 'home',
		pageUrl: 'https://example.com/',
		elementCount: 1
	};
	render(
		<MemoryRouter>
			<VerifyFindingDialog report={report} issue={issue} onClose={() => undefined} />
		</MemoryRouter>
	);
	expect(screen.queryByLabelText('Fresh ZIP build')).toBeNull();
	expect(screen.getByRole('alert').textContent).toContain(
		'Deploy the updated build to a public URL'
	);
	expect(screen.getByRole('button', { name: 'Run verification' }).hasAttribute('disabled')).toBe(
		true
	);
});
