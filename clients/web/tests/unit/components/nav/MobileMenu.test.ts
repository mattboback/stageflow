import { fireEvent, render } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

import MobileMenu from '$lib/components/nav/MobileMenu.svelte';

describe('MobileMenu', () => {
	it('renders dialog semantics and closes on Escape', async () => {
		const onClose = vi.fn();
		const { getByRole } = render(MobileMenu, {
			props: {
				isOpen: true,
				onClose,
				isActive: (_href: string) => false
			}
		});

		const dialog = getByRole('dialog', { name: 'Navigation menu' });
		await fireEvent.keyDown(dialog, { key: 'Escape' });
		expect(onClose).toHaveBeenCalledTimes(1);
	});
});
