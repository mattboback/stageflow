import PlaygroundPage from '$lib/components/playground/PlaygroundPage.svelte';
import { render } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

vi.mock('$lib/api/client', () => ({
	fetchScanners: vi.fn().mockResolvedValue({ scanners: [] }),
	getDefaultScannerSelections: vi.fn().mockReturnValue([]),
	submitScanJob: vi.fn()
}));

describe('PlaygroundPage', () => {
	it('renders the scan configuration surface and handles empty scanners', async () => {
		const { findByText, getByText } = render(PlaygroundPage);

		expect(getByText('Configure Your Scan')).toBeInTheDocument();
		expect(await findByText('No scanners available')).toBeInTheDocument();
		expect(getByText('Start Scan')).toBeInTheDocument();
	});
});
