import type { ScannerDefinition, ScannerSelection } from '../types/scan';
import { normalizeUrlInput, validateHttpUrl } from '../url';

export interface AuthFormConfig {
	enabled: boolean;
	loginUrl: string;
	username: string;
	password: string;
	usernameSelector: string;
	passwordSelector: string;
	submitSelector: string;
	successStrategy: 'auto' | 'selector';
	successSelector: string;
}

export function isAuthConfigComplete(config: AuthFormConfig): boolean {
	if (!config.enabled) {
		return true;
	}

	const loginUrl = normalizeUrlInput(config.loginUrl);
	return (
		loginUrl !== null &&
		validateHttpUrl(loginUrl).ok &&
		config.username.trim().length > 0 &&
		config.password.trim().length > 0 &&
		(config.successStrategy !== 'selector' || config.successSelector.trim().length > 0)
	);
}

export function buildFormAuthConfig(config: AuthFormConfig): Record<string, unknown> | null {
	if (!config.enabled || !isAuthConfigComplete(config)) {
		return null;
	}

	const usernameSelector = config.usernameSelector.trim() || 'auto:username';
	const passwordSelector = config.passwordSelector.trim() || 'auto:password';
	const submitSelector = config.submitSelector.trim() || 'auto:submit';

	const successSelector = config.successSelector.trim();
	const success =
		config.successStrategy === 'selector' && successSelector.length > 0
			? { type: 'selector', selector: successSelector }
			: { type: 'networkidle' };

	const loginUrl = normalizeUrlInput(config.loginUrl) ?? config.loginUrl.trim();

	return {
		mode: 'form',
		form: {
			login_url: loginUrl,
			steps: [
				{ type: 'fill', selector: usernameSelector, value: config.username },
				{ type: 'fill', selector: passwordSelector, value: config.password },
				{ type: 'click', selector: submitSelector }
			],
			success
		}
	};
}

export type PlaygroundMode = 'url' | 'zip';

export interface PlaygroundValidation {
	ready: boolean;
	message: string;
	error: string | null;
	focusId: string | null;
	validUrls: string[];
	urlRowErrors: Record<number, string>;
}

interface PlaygroundValidationInput {
	mode: PlaygroundMode;
	urls: string[];
	file: Pick<File, 'name' | 'size'> | null;
	selections: ScannerSelection[];
	auth: AuthFormConfig;
	// Explicit `| undefined` throughout: exactOptionalPropertyTypes separates an
	// absent property from one present-but-undefined, and callers build this
	// object from optional state rather than omitting keys.
	catalogLoading?: boolean | undefined;
	catalogError?: string | null | undefined;
	projectName?: string | null | undefined;
}

function invalidValidation(
	message: string,
	focusId: string | null,
	validUrls: string[] = [],
	urlRowErrors: Record<number, string> = {}
): PlaygroundValidation {
	return { ready: false, message, error: message, focusId, validUrls, urlRowErrors };
}

/** Single source of truth for the review summary, Run buttons, and submitted payload. */
export function validatePlaygroundConfiguration({
	mode,
	urls,
	file,
	selections,
	auth,
	catalogLoading = false,
	catalogError = null,
	projectName
}: PlaygroundValidationInput): PlaygroundValidation {
	if (catalogLoading) return invalidValidation('Loading the scanner catalog.', null);
	if (catalogError) return invalidValidation('The scanner catalog must load before running.', null);
	if (!selections.some((selection) => selection.enabled)) {
		return invalidValidation('Enable at least one scanner.', 'scanner-options');
	}
	if (projectName !== undefined && projectName !== null && !projectName.trim()) {
		return invalidValidation('Give this project a name before running a scan.', 'project-name');
	}

	const validUrls: string[] = [];
	const urlRowErrors: Record<number, string> = {};
	if (mode === 'url') {
		urls.forEach((raw, index) => {
			const normalized = normalizeUrlInput(raw);
			if (!normalized) return;
			const result = validateHttpUrl(normalized);
			if (result.ok) validUrls.push(result.url);
			else urlRowErrors[index] = result.reason;
		});
		const firstInvalidRow = Object.keys(urlRowErrors)
			.map(Number)
			.sort((a, b) => a - b)[0];
		if (firstInvalidRow !== undefined) {
			return invalidValidation(
				'Fix the invalid URL before running the scan.',
				`url-input-${firstInvalidRow}`,
				validUrls,
				urlRowErrors
			);
		}
		if (validUrls.length === 0) {
			return invalidValidation('Add at least one URL to scan.', 'url-input-0');
		}
	} else {
		if (!file) return invalidValidation('Choose a ZIP archive to scan.', 'zip-picker');
		const fileError = validateZipUploadFile(file);
		if (fileError) return invalidValidation(fileError, 'zip-picker');
	}

	if (mode === 'url' && auth.enabled) {
		if (!auth.loginUrl.trim())
			return invalidValidation('Enter the login URL.', 'auth-login-url', validUrls);
		const normalizedLoginUrl = normalizeUrlInput(auth.loginUrl);
		if (!normalizedLoginUrl || !validateHttpUrl(normalizedLoginUrl).ok) {
			return invalidValidation(
				'Enter a valid HTTP or HTTPS login URL.',
				'auth-login-url',
				validUrls
			);
		}
		if (!auth.username.trim())
			return invalidValidation('Enter the login username.', 'auth-username', validUrls);
		if (!auth.password.trim())
			return invalidValidation('Enter the login password.', 'auth-password', validUrls);
		if (auth.successStrategy === 'selector' && !auth.successSelector.trim()) {
			return invalidValidation(
				'Enter the selector that confirms login succeeded.',
				'auth-success-selector',
				validUrls
			);
		}
	}

	return {
		ready: true,
		message: "All set! You're ready to scan.",
		error: null,
		focusId: null,
		validUrls,
		urlRowErrors
	};
}

export interface RuntimeEstimate {
	label: string;
	detail: string;
}

function formatEstimateSeconds(seconds: number): string {
	if (seconds < 60) return `${seconds}s`;
	const minutes = Math.floor(seconds / 60);
	const remainder = seconds % 60;
	return remainder ? `${minutes}m ${remainder}s` : `${minutes}m`;
}

export function estimateScanRuntime(
	catalog: ScannerDefinition[],
	selections: ScannerSelection[],
	targetCount: number,
	mode: PlaygroundMode
): RuntimeEstimate {
	if (mode === 'zip') {
		return { label: 'Varies by archive', detail: 'Depends on pages found in the ZIP' };
	}
	if (targetCount <= 0 || !selections.some((selection) => selection.enabled)) {
		return { label: '—', detail: 'Add targets and scanners for an estimate' };
	}
	const catalogById = new Map(catalog.map((scanner) => [scanner.id, scanner]));
	const estimates = selections
		.filter((selection) => selection.enabled)
		.map((selection) => catalogById.get(selection.id)?.capabilities.estimatedTimePerPage ?? 0)
		.filter((value) => Number.isFinite(value) && value > 0);
	if (estimates.length === 0) {
		return { label: 'Varies by site', detail: 'The selected scanners provide no timing estimate' };
	}
	const slowestMs = Math.max(...estimates);
	const expectedSeconds = Math.max(1, Math.ceil((slowestMs * targetCount) / 1000));
	const lower = Math.max(1, Math.floor(expectedSeconds * 0.8));
	const upper = Math.max(lower + 1, Math.ceil(expectedSeconds * 1.5));
	return {
		label: `${formatEstimateSeconds(lower)}–${formatEstimateSeconds(upper)}`,
		detail: `${targetCount} ${targetCount === 1 ? 'page' : 'pages'}; scanners run in parallel`
	};
}

export function isZipFilename(name: string): boolean {
	return name.trim().toLowerCase().endsWith('.zip');
}

export const MAX_ZIP_UPLOAD_BYTES = 100 * 1024 * 1024;

export function validateZipUploadFile(file: Pick<File, 'name' | 'size'>): string | null {
	if (!isZipFilename(file.name)) {
		return 'Please select a ZIP file';
	}

	if (file.size > MAX_ZIP_UPLOAD_BYTES) {
		return 'ZIP file must be 100MB or smaller';
	}

	return null;
}
