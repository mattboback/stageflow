import type { ScannerDefinition, ScannerSelection, ScannersResponse } from '$lib/types/scan';

import { buildApiUrl } from './utils';

interface SubmitJobParams {
	mode: 'zip' | 'url';
	file: File | null;
	urls: string[];
	scanners: ScannerSelection[];
	highlightStyle: 'solid' | 'dashed';
	screenshot?: boolean;
	signal?: AbortSignal;
}

interface SubmitJobResponse {
	job_id: string;
	message?: string;
}

export async function submitScanJob({
	mode,
	file,
	urls,
	scanners,
	highlightStyle,
	screenshot = true,
	signal
}: SubmitJobParams): Promise<SubmitJobResponse> {
	let response: Response;

	// Enabled scanner module IDs
	const modules = scanners.filter((s) => s.enabled).map((s) => s.id);
	const scannerConfigs = scanners
		.filter((s) => s.enabled && s.config)
		.reduce<Record<string, unknown>>((acc, s) => {
			acc[s.id] = s.config;
			return acc;
		}, {});

	if (mode === 'zip') {
		if (!file) {
			throw new Error('Select a file');
		}
		const formData = new FormData();
		formData.append('file', file);
		formData.append('highlight_style', highlightStyle);
		formData.append('modules', modules.join(','));
		formData.append('screenshot', screenshot ? 'true' : 'false');
		if (Object.keys(scannerConfigs).length > 0) {
			formData.append('scanner_configs', JSON.stringify(scannerConfigs));
		}

		response = await fetch(buildApiUrl('/api/v1/jobs/zip'), {
			method: 'POST',
			body: formData,
			signal
		});
	} else {
		if (urls.length === 0) {
			throw new Error('Enter a URL');
		}
		response = await fetch(buildApiUrl('/api/v1/jobs/urls'), {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				urls,
				modules,
				scanner_configs: scannerConfigs,
				screenshot,
				highlight_style: highlightStyle
			}),
			signal
		});
	}

	const data = (await response.json().catch(() => null)) as SubmitJobResponse | null;
	if (!response.ok) {
		if (response.status === 413) {
			throw new Error('File too large. Maximum size is 100MB.');
		}
		if (response.status === 400) {
			throw new Error(data?.message ?? 'Invalid request. Please check your input.');
		}
		if (response.status === 422) {
			throw new Error(data?.message ?? 'Invalid scanner selection or URL format.');
		}
		if (response.status >= 500) {
			throw new Error('Server error. Please try again in a moment.');
		}
		throw new Error(data?.message ?? 'Scan failed. Please try again.');
	}
	if (!data?.job_id) {
		throw new Error('No job ID returned. Please try again.');
	}

	return data;
}

export async function fetchScanners(signal?: AbortSignal): Promise<ScannersResponse> {
	const response = await fetch(buildApiUrl('/api/v1/scanners'), {
		method: 'GET',
		headers: { 'Content-Type': 'application/json' },
		signal
	});

	if (!response.ok) {
		if (response.status >= 500) {
			throw new Error('Scanner service unavailable. Using default scanners.');
		}
		throw new Error('Failed to load scanners. Using default scanners.');
	}

	const data = (await response.json()) as ScannersResponse;

	// Hide disabled scanners from the UI by default.
	const enabledScanners = data.scanners.filter((scanner) => scanner.enabled);

	return {
		scanners: enabledScanners,
		categories: data.categories
	};
}

export function getDefaultScannerSelections(scanners: ScannerDefinition[]): ScannerSelection[] {
	const enabledScanners = scanners.filter((scanner) => scanner.enabled);
	const defaultScannerId = enabledScanners.some((scanner) => scanner.id === 'axe')
		? 'axe'
		: enabledScanners[0]?.id;

	return enabledScanners.map((scanner) => ({
		id: scanner.id,
		enabled: scanner.id === defaultScannerId
	}));
}
