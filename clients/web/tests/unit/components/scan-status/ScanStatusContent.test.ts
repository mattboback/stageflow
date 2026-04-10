import ScanStatusContent from '$lib/components/scan-status/ScanStatusContent.svelte';
import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

describe('ScanStatusContent', () => {
	it('renders processing view for non-terminal states', () => {
		render(ScanStatusContent, {
			props: {
				status: 'loading',
				result: null,
				logs: []
			}
		});

		expect(screen.getAllByText("What's happening now").length).toBeGreaterThan(0);
		expect(screen.getAllByText('Processing').length).toBeGreaterThan(0);
		expect(
			screen.getByText('Initializing the first scanner and preparing live progress.')
		).toBeInTheDocument();
	});

	it('renders scanner activity and the Lighthouse long-pole hint', () => {
		const { getByText } = render(ScanStatusContent, {
			props: {
				status: 'scanning',
				logs: [],
				result: {
					id: 'job-123',
					state: 'SCANNING',
					progress: {
						current_page: 1,
						total_pages: 3,
						percentage: 33
					},
					expected_scanners: ['axe', 'lighthouse'],
					completed_scanners: ['axe'],
					remaining_scanners: ['lighthouse'],
					created_at: new Date().toISOString(),
					updated_at: new Date().toISOString()
				}
			}
		});

		expect(getByText('Scanner Activity')).toBeInTheDocument();
		expect(getByText('Axe')).toBeInTheDocument();
		expect(getByText('Waiting on Lighthouse')).toBeInTheDocument();
	});

	it('renders completed and failed terminal views', () => {
		const completed = render(ScanStatusContent, {
			props: {
				status: 'complete',
				logs: [],
				result: {
					id: 'job-999',
					state: 'DONE',
					progress: {
						current_page: 2,
						total_pages: 2,
						percentage: 100
					},
					violations: 4,
					created_at: new Date().toISOString(),
					updated_at: new Date().toISOString()
				}
			}
		});

		expect(completed.getByText('Scan complete')).toBeInTheDocument();
		completed.unmount();

		const failed = render(ScanStatusContent, {
			props: {
				status: 'failed',
				logs: [],
				result: {
					id: 'job-998',
					state: 'FAILED',
					error: 'scanner crashed',
					last_stage: 'scanning',
					created_at: new Date().toISOString(),
					updated_at: new Date().toISOString()
				}
			}
		});

		expect(failed.getByText('Scan Failed')).toBeInTheDocument();
		expect(failed.getByText('scanner crashed')).toBeInTheDocument();
	});
});
