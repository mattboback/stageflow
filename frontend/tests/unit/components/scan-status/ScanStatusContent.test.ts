import ScanStatusContent from '$lib/components/scan-status/ScanStatusContent.svelte';
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

describe('ScanStatusContent', () => {
	it('renders processing view for non-terminal states', () => {
		const { getByText } = render(ScanStatusContent, {
			props: {
				status: 'loading',
				result: null,
				logs: []
			}
		});

		expect(getByText('Processing')).toBeInTheDocument();
		expect(getByText('Initializing scan environment...')).toBeInTheDocument();
	});
});
