import TerminalCardHeader from '$lib/components/ui/TerminalCardHeader.svelte';
import { cleanup, render } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';

describe('TerminalCardHeader', () => {
	afterEach(() => { cleanup(); });

	it('renders the path', () => {
		const { getByText } = render(TerminalCardHeader, { props: { path: '/projects/demo' } });
		expect(getByText('/projects/demo')).toBeInTheDocument();
	});

	it('renders badges when provided', () => {
		const { getByText } = render(TerminalCardHeader, {
			props: {
				path: '/projects/demo',
				badges: [
					{ label: 'Case Study', variant: 'terminal' },
					{ label: 'Featured', variant: 'status' }
				]
			}
		});

		expect(getByText('Case Study')).toBeInTheDocument();
		expect(getByText('Featured')).toBeInTheDocument();
	});
});
