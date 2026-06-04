import SeverityBreakdown from '$lib/components/report/SeverityBreakdown.svelte';
import { cleanup, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';

afterEach(() => {
	cleanup();
});

describe('SeverityBreakdown', () => {
	it('renders non-zero severity chips and proportional bars', () => {
		const { container } = render(SeverityBreakdown, {
			bySeverity: {
				critical: 2,
				serious: 1,
				moderate: 3,
				minor: 0,
				info: 4
			}
		});

		expect(screen.getByText('critical')).toBeInTheDocument();
		expect(screen.getByText('serious')).toBeInTheDocument();
		expect(screen.getByText('moderate')).toBeInTheDocument();
		expect(screen.getByText('info')).toBeInTheDocument();
		expect(screen.queryByText('minor')).not.toBeInTheDocument();

		const bars = container.querySelectorAll('.animate-grow');
		expect(bars).toHaveLength(4);
		expect(bars[0]).toHaveAttribute('style', expect.stringContaining('width: 20%'));
		expect(bars[3]).toHaveAttribute('style', expect.stringContaining('width: 40%'));
	});

	it('renders no chips or bars when all counts are zero', () => {
		const { container } = render(SeverityBreakdown, {
			bySeverity: {
				critical: 0,
				serious: 0,
				moderate: 0,
				minor: 0,
				info: 0
			}
		});

		expect(container.querySelectorAll('.animate-grow')).toHaveLength(0);
		expect(screen.queryByText('critical')).not.toBeInTheDocument();
		expect(screen.queryByText('serious')).not.toBeInTheDocument();
		expect(screen.queryByText('moderate')).not.toBeInTheDocument();
		expect(screen.queryByText('minor')).not.toBeInTheDocument();
		expect(screen.queryByText('info')).not.toBeInTheDocument();
	});
});
