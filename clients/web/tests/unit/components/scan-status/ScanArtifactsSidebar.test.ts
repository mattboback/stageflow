import ScanArtifactsSidebar from '$lib/components/scan-status/ScanArtifactsSidebar.svelte';
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

describe('ScanArtifactsSidebar', () => {
	it('shows locked-open state while scan is in progress', () => {
		const { getByRole, getByText } = render(ScanArtifactsSidebar, {
			props: {
				status: 'loading',
				result: null
			}
		});

		expect(getByText('Logs & Artifacts')).toBeInTheDocument();
		expect(getByText('Generating artifacts…')).toBeInTheDocument();

		const toggle = getByRole('button', { name: 'In Progress' });
		expect(toggle).toBeDisabled();
	});

	it('promotes the report as the primary next step after completion', () => {
		const { getByRole } = render(ScanArtifactsSidebar, {
			props: {
				status: 'complete',
				result: {
					id: 'job-1',
					state: 'COMPLETE',
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:00Z'
				}
			}
		});

		expect(getByRole('link', { name: /review unified report first/i })).toHaveAttribute(
			'href',
			'/scan/job-1/report'
		);
	});
});
