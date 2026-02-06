import type * as apiClient from '$lib/api/client';
import type { ScannerDefinition } from '$lib/types/scan';

import { fetchScanners } from '$lib/api/client';
import PlaygroundPage from '$lib/components/playground/PlaygroundPage.svelte';
import { cleanup, render, screen } from '@testing-library/svelte';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

vi.mock('$lib/api/client', async () => {
	const actual = await vi.importActual<typeof apiClient>('$lib/api/client');
	return {
		...actual,
		fetchScanners: vi.fn(),
		submitScanJob: vi.fn()
	};
});

const mockFetchScanners = vi.mocked(fetchScanners);

function createScanner(id: string): ScannerDefinition {
	return {
		id,
		name: id,
		version: '1.0.0',
		description: '',
		categories: [],
		aliases: [],
		enabled: true,
		builtIn: true,
		capabilities: {
			outputFormats: [],
			supportsScreenshots: false,
			supportsConcurrency: false,
			requiresBrowser: false,
			supportsOffline: false,
			maxConcurrency: 1
		}
	};
}

describe('PlaygroundPage', () => {
	beforeEach(() => {
		mockFetchScanners.mockReset();
	});
	afterEach(() => {
		cleanup();
	});

	it('renders the scan configuration surface and handles empty scanners', async () => {
		mockFetchScanners.mockResolvedValue({ scanners: [], categories: [] });

		render(PlaygroundPage);

		expect(screen.getByText('Configure Scan')).toBeInTheDocument();
		expect(await screen.findByText('No scanners available')).toBeInTheDocument();
		expect(screen.getByText('Start Scan')).toBeInTheDocument();
	});

	it('defaults to coverage preset and enables multiple scanners', async () => {
		mockFetchScanners.mockResolvedValue({
			scanners: [createScanner('axe'), createScanner('lighthouse'), createScanner('ai-navigator')],
			categories: []
		});

		render(PlaygroundPage);

		const coverageButton = await screen.findByRole('button', { name: 'Coverage' });
		expect(coverageButton).toHaveAttribute('aria-pressed', 'true');
		expect(screen.getByRole('button', { name: /axe/i })).toHaveAttribute('aria-pressed', 'true');
		expect(screen.getByRole('button', { name: /lighthouse/i })).toHaveAttribute(
			'aria-pressed',
			'true'
		);
		expect(screen.getByRole('button', { name: /ai navigator/i })).toHaveAttribute(
			'aria-pressed',
			'false'
		);
	});

	it('switching presets updates enabled count and manual toggle sets custom', async () => {
		mockFetchScanners.mockResolvedValue({
			scanners: [createScanner('axe'), createScanner('lighthouse'), createScanner('ai-navigator')],
			categories: []
		});

		const user = userEvent.setup();
		render(PlaygroundPage);

		await screen.findByRole('button', { name: 'Coverage' });
		await user.click(screen.getByRole('button', { name: 'Quick' }));
		expect(screen.getByRole('button', { name: /axe/i })).toHaveAttribute('aria-pressed', 'true');
		expect(screen.getByRole('button', { name: /lighthouse/i })).toHaveAttribute(
			'aria-pressed',
			'false'
		);

		await user.click(screen.getByRole('button', { name: /lighthouse/i }));
		const customButton = screen.getByRole('button', { name: 'Custom' });
		expect(customButton).toHaveAttribute('aria-pressed', 'true');
		expect(screen.getByRole('button', { name: /lighthouse/i })).toHaveAttribute(
			'aria-pressed',
			'true'
		);
	});
});
