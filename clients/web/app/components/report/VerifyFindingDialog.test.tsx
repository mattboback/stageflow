import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { loadAllScansFixture } from '../../test/load-fixture';
import { submitScanJob } from '../../lib/api/client';
import { saveVerificationRequest } from '../../lib/verification';
import type { IssueDetail, UnifiedReport } from '../../lib/types/unified-report';
import { VerifyFindingDialog } from './VerifyFindingDialog';

vi.mock('../../lib/api/client', () => ({ submitScanJob: vi.fn() }));
vi.mock('../../lib/verification', async (original) => ({
	...(await original<typeof import('../../lib/verification')>()),
	saveVerificationRequest: vi.fn()
}));

function source(
	inputType: 'urls' | 'zip',
	authenticated = false
): { report: UnifiedReport; issue: IssueDetail } {
	const report = loadAllScansFixture();
	const issue: IssueDetail = {
		id: 'issue',
		scanner: 'axe',
		ruleId: 'image-alt',
		title: 'Missing alt',
		description: 'Add alt text',
		severity: 'serious',
		pageId: 'page-v2-home',
		pageUrl: 'https://example.com/',
		elementCount: 1,
		occurrences: [{ elementId: 'occ-v2-logo', selector: '#logo' }]
	};
	report.meta.scanContext = {
		expectedPages: [issue.pageId],
		identityScope: 'path',
		scannerConfigFingerprints: { axe: 'config' },
		pageConfigFingerprints: { [issue.pageId]: 'page-config' },
		verificationEligiblePages: { [issue.pageId]: true },
		scannerConfigurations: { axe: { disabledRules: ['other-rule'] } },
		viewport: { width: 1280, height: 720 },
		purpose: 'full',
		authenticated,
		inputType
	};
	report.pages = [{ ...report.pages[0]!, id: issue.pageId, url: issue.pageUrl }];
	report.scanners = [
		{
			id: 'axe',
			name: 'Axe',
			status: 'success',
			toolVersion: '1.0',
			pages: [{ pageId: issue.pageId, status: 'success', finalUrl: issue.pageUrl }]
		}
	];
	return { report, issue };
}

beforeEach(() => {
	vi.clearAllMocks();
	vi.mocked(saveVerificationRequest).mockReturnValue(true);
	vi.mocked(submitScanJob).mockResolvedValue({ job_id: 'verification-job' });
});

describe('Verify this fix submission', () => {
	it('requires fresh authentication and cannot bypass it by removing the auth form', () => {
		render(
			<MemoryRouter>
				<VerifyFindingDialog {...source('urls', true)} onClose={() => undefined} />
			</MemoryRouter>
		);
		expect(screen.getByRole('button', { name: 'Run verification' }).hasAttribute('disabled')).toBe(
			true
		);
		fireEvent.click(screen.getByRole('button', { name: 'Remove' }));
		expect(screen.getByRole('button', { name: 'Run verification' }).hasAttribute('disabled')).toBe(
			true
		);
		expect(submitScanJob).not.toHaveBeenCalled();
	});

	it('uploads a fresh ZIP and targets only the affected page and responsible scanner', async () => {
		render(
			<MemoryRouter>
				<VerifyFindingDialog {...source('zip')} onClose={() => undefined} />
			</MemoryRouter>
		);
		expect(screen.getByRole('button', { name: 'Run verification' }).hasAttribute('disabled')).toBe(
			true
		);
		const file = new File(['fresh-build'], 'changed.zip', { type: 'application/zip' });
		fireEvent.change(screen.getByLabelText('Fresh ZIP build'), { target: { files: [file] } });
		fireEvent.click(screen.getByRole('button', { name: 'Run verification' }));
		await waitFor(() => expect(submitScanJob).toHaveBeenCalledOnce());
		const params = vi.mocked(submitScanJob).mock.calls[0]![0];
		expect(params.mode).toBe('zip');
		expect(params.file).toBe(file);
		expect(params.pagePaths).toEqual(['/']);
		expect(params.scanners).toEqual([
			{ id: 'axe', enabled: true, config: { disabledRules: ['other-rule'] } }
		]);
		expect(params.verification?.subject_id).toBe('occ-v2-logo');
	});

	it('does not create a scan when the original verification context cannot be saved', async () => {
		vi.mocked(saveVerificationRequest).mockReturnValue(false);
		render(
			<MemoryRouter>
				<VerifyFindingDialog {...source('urls')} onClose={() => undefined} />
			</MemoryRouter>
		);
		fireEvent.click(screen.getByRole('button', { name: 'Run verification' }));
		await screen.findByRole('alert');
		expect(screen.getByRole('alert').textContent).toContain('Browser storage is unavailable');
		expect(submitScanJob).not.toHaveBeenCalled();
	});
});
