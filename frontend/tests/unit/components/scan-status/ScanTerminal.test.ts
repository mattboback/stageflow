import ScanTerminal from '$lib/components/scan-status/ScanTerminal.svelte';
import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';

describe('ScanTerminal', () => {
	it('shows connecting message when logs are empty', () => {
		const { getByText } = render(ScanTerminal, {
			props: {
				logs: []
			}
		});

		expect(getByText('Waiting for live scan events...')).toBeInTheDocument();
	});
});
