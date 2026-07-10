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

	return (
		config.loginUrl.trim().length > 0 &&
		config.username.trim().length > 0 &&
		config.password.trim().length > 0
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
	model: 'anthropic/claude-3.5-sonnet',
	inputValues: [],
	successCriteria: []
};

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

export function validateZipUploadFile(file: File): string | null {
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
