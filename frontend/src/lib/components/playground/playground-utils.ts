export interface AiInputValue {
	key: string;
	value: string;
}

export interface AiSuccessCriterion {
	type: string;
	value: string;
}

export function normalizeUrlInput(input: string): string | null {
	const trimmed = input.trim();
	if (!trimmed) return null;

	// If the user provided an explicit scheme (http://, https://, etc), respect it.
	if (/^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(trimmed)) {
		return trimmed;
	}

	// Support protocol-relative URLs (e.g. //example.com).
	if (trimmed.startsWith('//')) {
		return `https:${trimmed}`;
	}

	// Default to HTTPS when scheme is omitted.
	return `https://${trimmed}`;
}

export function normalizeUrlListText(input: string): { text: string; changed: boolean } {
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
				invalid.push({ url, reason: 'URL must start with http:// or https://.' });
				continue;
			}
			if (!parsed.hostname) {
				invalid.push({ url, reason: 'Missing hostname.' });
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
