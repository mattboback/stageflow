import type { ScannerDefinition, ScannerSelection, ScannersResponse } from '$lib/types/scan';

import { applyScannerPreset } from '$lib/components/playground/scanner-presets';

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

interface ApiErrorDetail {
	code?: string;
	message?: string;
	details?: string;
	suggestion?: string;
	field?: string;
}

interface ApiErrorResponse {
	message?: string;
	error?: ApiErrorDetail;
}

function readApiErrorMessage(data: ApiErrorResponse | null): string | null {
	if (!data) {
		return null;
	}

	const topLevel = data.message?.trim();
	if (topLevel) {
		return topLevel;
	}

	const detail = data.error;
	if (!detail) {
		return null;
	}

	const message = detail.message?.trim();
	const suggestion = detail.suggestion?.trim();
	const details = detail.details?.trim();

	if (!message) {
		return null;
	}

	const extras = [suggestion, details].filter(
		(value): value is string => Boolean(value && value.length > 0)
	);

	return extras.length > 0 ? `${message} ${extras.join(' ')}` : message;
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

	const data = (await response.json().catch(() => null)) as
		| (SubmitJobResponse & ApiErrorResponse)
		| null;
	const serverMessage = readApiErrorMessage(data);
	if (!response.ok) {
		if (response.status === 413) {
			throw new Error('File too large. Maximum size is 100MB.');
		}
		if (response.status === 400) {
			throw new Error(serverMessage ?? 'Invalid request. Please check your input.');
		}
		if (response.status === 422) {
			throw new Error(serverMessage ?? 'Invalid scanner selection or URL format.');
		}
		if (response.status >= 500) {
			throw new Error(serverMessage ?? 'Server error. Please try again in a moment.');
		}
		throw new Error(serverMessage ?? 'Scan failed. Please try again.');
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
	if (enabledScanners.length === 0) {
		return [];
	}

	const selections = enabledScanners.map((scanner) => ({
		id: scanner.id,
		enabled: false
	}));

	return applyScannerPreset(selections, 'coverage');
}
