import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { describe, expect, it } from 'vitest';
import { loadAllScansFixture } from '../../test/load-fixture';
import type { VerificationRequest } from '../../lib/verification';
import { VerificationResultBanner } from './VerificationResultBanner';

const request: VerificationRequest = {
	jobId: 'targeted',
	sourceJobId: 'original',
	createdAt: '2026-09-30T00:00:00Z',
	issue: {
		id: 'image',
		title: 'Missing image alternative',
		scanner: 'axe',
		ruleId: 'image-alt',
		pageId: 'page',
		pageUrl: 'https://sample.invalid/'
	},
	sourceContext: {
		expectedPages: ['page'],
		identityScope: 'path',
		inputType: 'zip',
		authenticated: false,
		purpose: 'full',
		scannerConfigFingerprints: { axe: 'settings' }
	},
	sourceScannerVersion: '0.2.0',
	expectedFinalUrl: 'http://127.0.0.1:8080/index.html'
};

describe('VerificationResultBanner', () => {
	it('shows pending targeted evaluation without a report', () => {
		render(
			<MemoryRouter>
				<VerificationResultBanner
					jobId="targeted"
					status="scanning"
					report={null}
					context={request}
				/>
			</MemoryRouter>
		);
		expect(screen.getByRole('heading', { name: 'Targeted verification: Pending' })).toBeTruthy();
	});
	it.each(['failed', 'error', 'cancelled', 'complete'] as const)(
		'shows unverified when %s produces no report',
		(status) => {
			render(
				<MemoryRouter>
					<VerificationResultBanner jobId="targeted" status={status} context={request} />
				</MemoryRouter>
			);
			expect(
				screen.getByRole('heading', { name: 'Targeted verification: Unverified' })
			).toBeTruthy();
			expect(
				screen.getByRole('link', { name: 'Return to the original report' }).getAttribute('href')
			).toBe('/scan/original/report');
		}
	);
	it('does not attach unrelated persisted descriptors to ordinary jobs', () => {
		render(
			<MemoryRouter>
				<VerificationResultBanner jobId="other" status="scanning" context={request} />
			</MemoryRouter>
		);
		expect(screen.queryByRole('region', { name: 'Targeted verification result' })).toBeNull();
	});
	it('keeps failed scoped reports explicitly unverified without successful scanner context', () => {
		const report = loadAllScansFixture();
		report.meta.jobId = 'targeted';
		report.meta.scanContext = {
			expectedPages: ['page'],
			purpose: 'verification',
			identityScope: 'path',
			inputType: 'zip',
			authenticated: false,
			scannerConfigFingerprints: {}
		};
		report.scanners = [{ id: 'axe', status: 'failed' }];
		render(
			<MemoryRouter>
				<VerificationResultBanner report={report} context={request} />
			</MemoryRouter>
		);
		expect(screen.getByRole('heading', { name: 'Targeted verification: Unverified' })).toBeTruthy();
	});
});
