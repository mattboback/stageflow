export function isPlainObject(
	value: unknown,
): value is Record<string, unknown> {
	if (!value) {
		return false;
	}

	if (typeof value !== "object") {
		return false;
	}

	if (Array.isArray(value)) {
		return false;
	}

	return true;
}

export function parseFirstJsonObject(
	text: string,
): Record<string, unknown> | undefined {
	const trimmed = text.trim();
	if (!trimmed) {
		return undefined;
	}

	const direct = tryParseJson(trimmed);
	if (isPlainObject(direct)) {
		return direct;
	}

	const fenced = extractFencedBlock(trimmed);
	if (fenced) {
		const parsed = tryParseJson(fenced);
		if (isPlainObject(parsed)) {
			return parsed;
		}
	}

	const braceCandidate = extractBraceCandidate(trimmed);
	if (!braceCandidate) {
		return undefined;
	}

	const parsed = tryParseJson(braceCandidate);
	if (!isPlainObject(parsed)) {
		return undefined;
	}

	return parsed;
}

function tryParseJson(raw: string): unknown {
	try {
		return JSON.parse(raw) as unknown;
	} catch {
		return undefined;
	}
}

function extractFencedBlock(text: string): string | undefined {
	const fencePattern = /```(?:json)?\s*([\s\S]*?)\s*```/i;
	const match = fencePattern.exec(text);
	if (!match) {
		return undefined;
	}

	return match[1]?.trim();
}

function extractBraceCandidate(text: string): string | undefined {
	const first = text.indexOf("{");
	if (first < 0) {
		return undefined;
	}

	const last = text.lastIndexOf("}");
	if (last <= first) {
		return undefined;
	}

	return text.slice(first, last + 1);
}
