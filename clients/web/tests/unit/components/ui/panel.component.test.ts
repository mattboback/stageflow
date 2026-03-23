import { cleanup, render } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';

import PanelFixture from '../../../fixtures/PanelFixture.svelte';

describe('Panel', () => {
	afterEach(() => {
		cleanup();
	});

	it('renders children', () => {
		const { getByTestId } = render(PanelFixture);
		expect(getByTestId('panel-child')).toBeInTheDocument();
	});

	it('applies variants', () => {
		const { getByTestId } = render(PanelFixture, {
			props: {
				variant: 'muted',
				padding: 'xs',
				rounded: '3xl',
				interactive: true
			}
		});
		const panel = getByTestId('panel');

		expect(panel).toHaveClass('bg-surface-muted');
		expect(panel).toHaveClass('p-2');
		expect(panel).toHaveClass('rounded-3xl');
		expect(panel).toHaveClass('hover-glow');
	});
});
