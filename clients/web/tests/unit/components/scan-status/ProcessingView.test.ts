import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import ProcessingView from '$lib/components/scan-status/ProcessingView.svelte';

describe('ProcessingView', () => {
	it('formats scanner names from scanner ids without hardcoded labels', () => {
		const { getByText } = render(ProcessingView, {
			props: {
				result: {
					id: 'job-1',
					state: 'SCANNING',
					progress: { current_page: 1, total_pages: 2, percentage: 50 },
					completed_scanners: ['axe'],
					remaining_scanners: ['ai-navigator'],
					created_at: '2026-01-01T00:00:00Z',
					updated_at: '2026-01-01T00:00:00Z'
				},
				logs: []
			}
		});

		expect(getByText('Ai Navigator')).toBeInTheDocument();
		expect(getByText('Axe')).toBeInTheDocument();
	});
});
