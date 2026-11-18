import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import type { ScanResult } from '../../lib/types/scan';
import { ArtifactsView } from './ArtifactsView';

const job: ScanResult = {
	id: 'job-1',
	state: 'DONE',
	created_at: '2026-10-02T00:00:00Z',
	updated_at: '2026-10-02T00:01:00Z',
	artifacts: {
		report_json: '/api/v1/jobs/job-1/files/report.json',
		scanner_artifacts: {
			axe: { scanner_type: 'axe', results_json: '/api/v1/jobs/job-1/files/axe/results.json' },
			seo: { scanner_type: 'seo' }
		}
	}
};

function hrefs(): string[] {
	return screen.getAllByRole('link').map((link) => link.getAttribute('href') ?? '');
}

describe('ArtifactsView', () => {
	it('links only files the API still serves', () => {
		render(<ArtifactsView jobId="job-1" job={job} />);

		const links = hrefs();
		expect(links).toHaveLength(2);
		expect(links[0]).toMatch(/\/api\/v1\/jobs\/job-1\/results$/);
		expect(links[1]).toBe('/api/v1/jobs/job-1/files/axe/results.json');
		// The per-scanner HTML report and its route were deleted; a link to it 404s.
		expect(links.some((href) => href.endsWith('/report'))).toBe(false);
	});

	it('shows a scanner without results as unavailable rather than linking it', () => {
		render(<ArtifactsView jobId="job-1" job={job} />);

		const row = screen.getByRole('row', { name: /SEO/ });
		expect(row.querySelector('a')).toBeNull();
		expect(row.textContent).toContain('—');
	});

	it('offers nothing to download for an archived run with no job', () => {
		render(<ArtifactsView jobId="" job={null} />);

		expect(screen.queryAllByRole('link')).toHaveLength(0);
		expect(screen.getByText('No aggregated report yet.')).toBeTruthy();
		expect(screen.getByText('No scanner results available yet.')).toBeTruthy();
	});
});
