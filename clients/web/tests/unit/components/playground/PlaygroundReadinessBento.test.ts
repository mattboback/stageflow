import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import PlaygroundReadinessBentoHarness from './PlaygroundReadinessBentoHarness.svelte';

describe('PlaygroundReadinessBento', () => {
	it('renders bento frame with main grid and bottom strip', () => {
		const { container } = render(PlaygroundReadinessBentoHarness);

		expect(container.querySelector('.bento-frame')).toBeInTheDocument();
		expect(container.querySelector('.bento-card')).toBeInTheDocument();
		expect(container.querySelector('.bento-grid-main')).toBeInTheDocument();
		expect(container.querySelector('.bento-grid-bottom')).toBeInTheDocument();
	});

	it('renders nav label and scanner count', () => {
		const { getAllByText } = render(PlaygroundReadinessBentoHarness);

		expect(getAllByText('Configure Scan').length).toBeGreaterThan(0);
		expect(getAllByText('3 scanners').length).toBeGreaterThan(0);
	});

	it('renders left and right panel content', () => {
		const { getAllByText } = render(PlaygroundReadinessBentoHarness);

		expect(getAllByText('Left panel content').length).toBeGreaterThan(0);
		expect(getAllByText('Right panel content').length).toBeGreaterThan(0);
	});

	it('renders bottom strip content', () => {
		const { getAllByText } = render(PlaygroundReadinessBentoHarness);

		expect(getAllByText('Bottom left').length).toBeGreaterThan(0);
		expect(getAllByText('Bottom center').length).toBeGreaterThan(0);
		expect(getAllByText('Bottom right').length).toBeGreaterThan(0);
	});

	it('shows ready label when provided', () => {
		const { getAllByText } = render(PlaygroundReadinessBentoHarness);

		expect(getAllByText('Ready').length).toBeGreaterThan(0);
	});
});
