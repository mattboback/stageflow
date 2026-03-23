import { cleanup, render } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';

import ChipFixture from '../../../fixtures/ChipFixture.svelte';

describe('Chip', () => {
	afterEach(() => {
		cleanup();
	});

	it('renders as a span by default', () => {
		const { getByTestId } = render(ChipFixture);
		const el = getByTestId('chip-span');
		expect(el.tagName.toLowerCase()).toBe('span');
	});

	it('can render as a button', () => {
		const { getByTestId } = render(ChipFixture);
		const el = getByTestId('chip-button');
		expect(el.tagName.toLowerCase()).toBe('button');
	});
});
