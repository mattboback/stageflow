import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import ScanStatusLiveBentoHarness from './ScanStatusLiveBentoHarness.svelte';

describe('ScanStatusLiveBento', () => {
	it('renders bento frame with main grid and bottom strip', () => {
		const { container } = render(ScanStatusLiveBentoHarness);

		expect(container.querySelector('.bento-frame')).toBeInTheDocument();
		expect(container.querySelector('.bento-card')).toBeInTheDocument();
		expect(container.querySelector('.bento-grid-main')).toBeInTheDocument();
		expect(container.querySelector('.bento-grid-bottom')).toBeInTheDocument();
	});

	it('renders header, main, and sidebar content', () => {
		const { getAllByText } = render(ScanStatusLiveBentoHarness);

		expect(getAllByText('Header content').length).toBeGreaterThan(0);
		expect(getAllByText('Main content').length).toBeGreaterThan(0);
		expect(getAllByText('Sidebar content').length).toBeGreaterThan(0);
	});

	it('shows status label and elapsed time', () => {
		const { getAllByText } = render(ScanStatusLiveBentoHarness);

		expect(getAllByText('Running').length).toBeGreaterThan(0);
		expect(getAllByText('45s').length).toBeGreaterThan(0);
	});

	it('shows scanner count', () => {
		const { getAllByText } = render(ScanStatusLiveBentoHarness);

		expect(getAllByText('6').length).toBeGreaterThan(0);
	});
});
