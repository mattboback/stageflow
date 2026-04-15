import type * as apiClient from '$lib/api/client';
import type { ScannerDefinition } from '$lib/types/scan';

import { fetchScanners, submitScanJob } from '$lib/api/client';
import PlaygroundPage from '$lib/components/playground/PlaygroundPage.svelte';
import { cleanup, render, screen, waitFor } from '@testing-library/svelte';
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
const mockSubmitScanJob = vi.mocked(submitScanJob);

function expectTextarea(element: Element | null): HTMLTextAreaElement {
	if (!(element instanceof HTMLTextAreaElement)) {
		throw new Error('Expected URL input to render as a textarea');
	}

	return element;
}

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
		mockSubmitScanJob.mockReset();
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
		expect(screen.getByText('What happens next')).toBeInTheDocument();
		expect(screen.getByText(/live scan status/i)).toBeInTheDocument();
	});

	it('uses multiline placeholder and shows normalization guidance', async () => {
		mockFetchScanners.mockResolvedValue({
			scanners: [createScanner('axe'), createScanner('lighthouse'), createScanner('ai-navigator')],
			categories: []
		});

		render(PlaygroundPage);

		const textarea = expectTextarea(await screen.findByLabelText('URLs to Scan'));
		expect(textarea.placeholder).toBe(
			'https://example.com\nexample.com/pricing\nexample.com/contact'
		);
		expect(screen.getByText(/enter one URL per line/i)).toBeInTheDocument();
		expect(screen.getByText(/normalized to `https:\/\/`/i)).toBeInTheDocument();
	});

	it('normalizes scheme-less input on blur', async () => {
		mockFetchScanners.mockResolvedValue({
			scanners: [createScanner('axe'), createScanner('lighthouse'), createScanner('ai-navigator')],
			categories: []
		});

		const user = userEvent.setup();
		render(PlaygroundPage);

		const textarea = expectTextarea(await screen.findByLabelText('URLs to Scan'));
		await user.type(textarea, 'example.com');
		await user.tab();

		await waitFor(() => {
			expect(textarea.value).toBe('https://example.com');
		});
	});

	it('submits silently normalized urls', async () => {
		mockFetchScanners.mockResolvedValue({
			scanners: [createScanner('axe'), createScanner('lighthouse'), createScanner('ai-navigator')],
			categories: []
		});
		mockSubmitScanJob.mockResolvedValue({ job_id: 'job-123' });

		const user = userEvent.setup();
		render(PlaygroundPage);

		const textarea = expectTextarea(await screen.findByLabelText('URLs to Scan'));
		await user.type(textarea, 'example.com');
		await user.click(screen.getByRole('button', { name: 'Start Scan' }));

		await waitFor(() => {
			expect(mockSubmitScanJob).toHaveBeenCalledWith(
				expect.objectContaining({
					mode: 'url',
					urls: ['https://example.com']
				})
			);
		});
	});

	it('defaults to coverage preset and enables multiple scanners', async () => {
		mockFetchScanners.mockResolvedValue({
			scanners: [createScanner('axe'), createScanner('lighthouse'), createScanner('ai-navigator')],
			categories: []
		});

		render(PlaygroundPage);

		const coverageButton = await screen.findByRole('button', {
			name: 'Coverage'
		});
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
	}, 15000);

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
	}, 15000);

	it('adds accessible labels to AI navigator dynamic fields', async () => {
		mockFetchScanners.mockResolvedValue({
			scanners: [createScanner('axe'), createScanner('lighthouse'), createScanner('ai-navigator')],
			categories: []
		});

		const user = userEvent.setup();
		render(PlaygroundPage);

		await screen.findByRole('button', { name: 'Coverage' });
		await user.click(screen.getByRole('button', { name: /ai navigator/i }));
		await user.click(screen.getByRole('button', { name: 'Add Field' }));

		expect(screen.getByLabelText('Input field name 1')).toHaveAttribute(
			'name',
			'aiInputValues[0].key'
		);
		expect(screen.getByLabelText('Input field value 1')).toHaveAttribute(
			'name',
			'aiInputValues[0].value'
		);
		expect(screen.getByRole('button', { name: 'Remove input field 1' })).toBeInTheDocument();

		await user.click(screen.getByRole('button', { name: 'Advanced Settings' }));
		await user.click(screen.getByRole('button', { name: 'Add Criterion' }));
		expect(screen.getByLabelText('Success criterion type 1')).toHaveAttribute(
			'name',
			'aiSuccessCriteria[0].type'
		);
		expect(screen.getByLabelText('Success criterion value 1')).toHaveAttribute(
			'name',
			'aiSuccessCriteria[0].value'
		);
		expect(screen.getByRole('button', { name: 'Remove success criterion 1' })).toBeInTheDocument();
	}, 15000);

	it('renders ZIP upload as a native button surface', async () => {
		mockFetchScanners.mockResolvedValue({
			scanners: [createScanner('axe'), createScanner('lighthouse'), createScanner('ai-navigator')],
			categories: []
		});

		const user = userEvent.setup();
		const { container } = render(PlaygroundPage);

		await screen.findByRole('button', { name: 'Coverage' });
		await user.click(screen.getByRole('button', { name: /zip archive/i }));

		const uploadButton = screen.getByRole('button', { name: 'Choose a ZIP file to upload' });
		expect(uploadButton).toBeInTheDocument();
		expect(uploadButton.tagName.toLowerCase()).toBe('button');
		expect(container.querySelector('input[type="file"]')).toHaveAttribute(
			'name',
			'staticSiteArchive'
		);
	}, 15000);
});
