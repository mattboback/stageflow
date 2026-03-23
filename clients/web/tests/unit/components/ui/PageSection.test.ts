import { cleanup, render } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';

import PageSectionFixture from '../../../fixtures/PageSectionFixture.svelte';

describe('PageSection', () => {
	afterEach(() => {
		cleanup();
	});

	it('applies default padding', () => {
		const { getByTestId } = render(PageSectionFixture);
		const section = getByTestId('page-section');

		expect(section).toHaveClass('pt-24');
		expect(section).toHaveClass('pb-20');
	});

	it('applies page padding', () => {
		const { getByTestId } = render(PageSectionFixture, {
			props: { padding: 'page' }
		});
		const section = getByTestId('page-section');

		expect(section).toHaveClass('pt-28');
		expect(section).toHaveClass('pb-24');
	});

	it('renders container wrapper by default', () => {
		const { container, getByTestId } = render(PageSectionFixture);

		expect(getByTestId('page-section-child')).toBeInTheDocument();
		expect(container.querySelector('.container-width')).toBeInTheDocument();
	});

	it('can disable the container wrapper', () => {
		const { container } = render(PageSectionFixture, {
			props: { disableContainer: true }
		});

		expect(container.querySelector('.container-width')).toBeNull();
	});
});
