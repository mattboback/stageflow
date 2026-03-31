import { fireEvent, render } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';

import Tabs from '$lib/components/ui/Tabs.svelte';

describe('Tabs', () => {
	it('renders tablist semantics and keyboard navigation', async () => {
		const onValueChange = vi.fn();
		const { getAllByRole } = render(Tabs, {
			props: {
				tabs: [
					{ id: 'one', label: 'One' },
					{ id: 'two', label: 'Two' },
					{ id: 'three', label: 'Three' }
				],
				defaultTab: 'one',
				onValueChange
			}
		});

		const tabs = getAllByRole('tab');
		expect(tabs).toHaveLength(3);
		expect(tabs[0]).toHaveAttribute('aria-selected', 'true');

		await fireEvent.keyDown(tabs[0], { key: 'ArrowRight' });
		expect(onValueChange).toHaveBeenCalledWith('two');
	});
});
