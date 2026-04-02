import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

import Progress from '$lib/components/ui/Progress.svelte';

describe('Progress', () => {
	it('exposes progressbar aria attributes', () => {
		const { getByRole } = render(Progress, {
			props: {
				value: 25,
				max: 50,
				ariaLabel: 'Scan progress'
			}
		});

		const progressbar = getByRole('progressbar', { name: 'Scan progress' });
		expect(progressbar).toHaveAttribute('aria-valuemin', '0');
		expect(progressbar).toHaveAttribute('aria-valuemax', '50');
		expect(progressbar).toHaveAttribute('aria-valuenow', '25');
	});
});
