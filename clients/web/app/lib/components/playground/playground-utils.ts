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
		validateHttpUrls([loginUrl]).invalid.length === 0 &&
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

export interface AiInputValue {
	key: string;
	value: string;
}

export interface AiSuccessCriterion {
	type: string;
	value: string;
}

export interface AiConfigState {
	objective: string;
	maxSteps: number;
	maxWallTimeMs: number;
	model: string;
	inputValues: AiInputValue[];
	successCriteria: AiSuccessCriterion[];
}

export const DEFAULT_AI_CONFIG: AiConfigState = {
	objective: '',
	maxSteps: 12,
	maxWallTimeMs: 120_000,
	model: AI_NAVIGATOR_DEFAULT_MODEL,
	inputValues: [],
	successCriteria: []
};

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
	ai: AiConfigState;
	aiEnabled: boolean;
	catalogLoading?: boolean;
	catalogError?: string | null;
	projectName?: string | null;
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
	ai,
	aiEnabled,
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
			const result = validateHttpUrls([normalized]);
			if (result.invalid[0]) urlRowErrors[index] = result.invalid[0].reason;
			else if (result.valid[0]) validUrls.push(result.valid[0]);
		});
		const firstInvalidRow = Object.keys(urlRowErrors).map(Number).sort((a, b) => a - b)[0];
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
		if (!auth.loginUrl.trim()) return invalidValidation('Enter the login URL.', 'auth-login-url', validUrls);
		const normalizedLoginUrl = normalizeUrlInput(auth.loginUrl);
		if (!normalizedLoginUrl || validateHttpUrls([normalizedLoginUrl]).invalid.length > 0) {
			return invalidValidation('Enter a valid HTTP or HTTPS login URL.', 'auth-login-url', validUrls);
		}
		if (!auth.username.trim()) return invalidValidation('Enter the login username.', 'auth-username', validUrls);
		if (!auth.password.trim()) return invalidValidation('Enter the login password.', 'auth-password', validUrls);
		if (auth.successStrategy === 'selector' && !auth.successSelector.trim()) {
			return invalidValidation('Enter the selector that confirms login succeeded.', 'auth-success-selector', validUrls);
		}
	}

	if (aiEnabled) {
		if (!ai.objective.trim()) return invalidValidation('Enter an AI Navigator objective.', 'ai-objective', validUrls);
		if (!ai.model.trim()) return invalidValidation('Enter an AI Navigator model.', 'ai-model', validUrls);
		if (!Number.isInteger(ai.maxSteps) || ai.maxSteps < 1 || ai.maxSteps > 50) {
			return invalidValidation('AI Navigator max steps must be between 1 and 50.', 'ai-max-steps', validUrls);
		}
		if (!Number.isFinite(ai.maxWallTimeMs) || ai.maxWallTimeMs < 10_000 || ai.maxWallTimeMs > 600_000) {
			return invalidValidation('AI Navigator wall time must be between 10,000 and 600,000 ms.', 'ai-wall-time', validUrls);
		}
		for (const [index, input] of ai.inputValues.entries()) {
			if (input.key.trim() && !input.value.trim()) {
				return invalidValidation('Complete or remove the empty AI input value.', `ai-input-value-${index}`, validUrls);
			}
			if (!input.key.trim() && input.value.trim()) {
				return invalidValidation('Complete or remove the empty AI input key.', `ai-input-key-${index}`, validUrls);
			}
		}
		for (const [index, criterion] of ai.successCriteria.entries()) {
			if (!criterion.value.trim()) {
				return invalidValidation('Complete or remove the empty success criterion.', `ai-criterion-value-${index}`, validUrls);
			}
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

export function normalizeUrlInput(input: string): string | null {
	const trimmed = input.trim();
	if (!trimmed) return null;

	if (/^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(trimmed)) {
		return trimmed;
	}

	if (trimmed.startsWith('//')) {
		return `https:${trimmed}`;
	}

	return `https://${trimmed}`;
}

export function normalizeUrlListText(input: string): {
	text: string;
	changed: boolean;
} {
	const normalized = input
		.split('\n')
		.map((line) => normalizeUrlInput(line))
		.filter((line): line is string => Boolean(line))
		.join('\n');

	const originalNormalized = input
		.split('\n')
		.map((line) => line.trim())
		.filter(Boolean)
		.join('\n');

	return { text: normalized, changed: normalized !== originalNormalized };
}

export function parseUrlList(input: string): string[] {
	return input
		.split('\n')
		.map((line) => normalizeUrlInput(line))
		.filter((line): line is string => Boolean(line));
}

export function validateHttpUrls(urls: string[]): {
	valid: string[];
	invalid: { url: string; reason: string }[];
} {
	const valid: string[] = [];
	const invalid: { url: string; reason: string }[] = [];

	for (const url of urls) {
		try {
			const parsed = new URL(url);
			const protocol = parsed.protocol.toLowerCase();
			if (protocol !== 'http:' && protocol !== 'https:') {
				invalid.push({
					url,
					reason: 'URL must start with http:// or https://.'
				});
				continue;
			}
			const hostname = parsed.hostname;
			if (!hostname) {
				invalid.push({ url, reason: 'Missing hostname.' });
				continue;
			}
			const hasDot = hostname.includes('.');
			const isLocalhost = hostname.toLowerCase() === 'localhost';
			// Node may keep brackets; browsers expose bare IPv6 (colons, no dots).
			const isIpv6 =
				(hostname.startsWith('[') && hostname.endsWith(']')) ||
				(!hasDot && hostname.includes(':'));
			if (!hasDot && !isLocalhost && !isIpv6) {
				invalid.push({
					url,
					reason: 'Hostname must contain a dot or be localhost.'
				});
				continue;
			}
			valid.push(url);
		} catch {
			invalid.push({ url, reason: 'Invalid URL.' });
		}
	}

	return { valid, invalid };
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

export function buildAiNavigatorConfig(params: {
	objective: string;
	maxSteps: number;
	maxWallTimeMs: number;
	model: string;
	inputValues: AiInputValue[];
	successCriteria: AiSuccessCriterion[];
}): Record<string, unknown> {
	const config: Record<string, unknown> = {
		goal: {
			objective: params.objective.trim(),
			maxSteps: params.maxSteps,
			maxWallTimeMs: params.maxWallTimeMs
		},
		vision: {
			provider: 'openrouter',
			model: params.model
		}
	};

	const inputValuesObj = params.inputValues
		.filter((item) => item.key.trim() && item.value.trim())
		.reduce<Record<string, string>>((acc, item) => {
			acc[item.key.trim()] = item.value.trim();
			return acc;
		}, {});

	if (Object.keys(inputValuesObj).length > 0) {
		(config.goal as Record<string, unknown>).inputValues = inputValuesObj;
	}

	const criteria = params.successCriteria
		.filter((item) => item.type && item.value.trim())
		.map((item) => ({ type: item.type, value: item.value.trim() }));

	if (criteria.length > 0) {
		(config.goal as Record<string, unknown>).successCriteria = criteria;
	}

	return config;
}
import { AI_NAVIGATOR_DEFAULT_MODEL } from '../../site-metadata';
import type { ScannerDefinition, ScannerSelection } from '../../types/scan';
