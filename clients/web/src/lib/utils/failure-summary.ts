export interface HeadingOrderContext {
	previousLevel: number;
	currentLevel: number;
}

const HEADER_LINES = new Set([
	"fix any of the following",
	"fix all of the following",
	"fix the following",
	"fix one of the following",
]);

const normalizeLine = (line: string) =>
	line.trim().replace(/:$/, "").toLowerCase();

const stripBullet = (line: string) =>
	line
		.replace(/^[-*\u2022]\s+/, "")
		.replace(/^\d+\.?\s+/, "")
		.trim();

export const extractFailureDetails = (summary?: string | null): string[] => {
	if (!summary) return [];

	const lines = summary
		.split("\n")
		.map((line) => line.trim())
		.filter(Boolean);

	const details = lines
		.filter((line) => !HEADER_LINES.has(normalizeLine(line)))
		.map(stripBullet)
		.filter(Boolean);

	return details.length > 0 ? details : lines;
};

export const extractPrimaryFailureDetail = (
	summary?: string | null,
): string | null => {
	const details = extractFailureDetails(summary);
	return details[0] ?? null;
};

export const parseHeadingOrderContext = (
	detail?: string | null,
): HeadingOrderContext | null => {
	if (!detail) return null;

	const match = /([hH][1-6])\s+follows\s+([hH][1-6])/i.exec(detail);
	if (!match) return null;

	const currentLevel = Number.parseInt(match[1][1], 10);
	const previousLevel = Number.parseInt(match[2][1], 10);

	if (!Number.isFinite(currentLevel) || !Number.isFinite(previousLevel)) {
		return null;
	}

	return { previousLevel, currentLevel };
};

export const formatHeadingOrderFlow = (
	context: HeadingOrderContext,
): string => {
	const base = `H${context.previousLevel} -> H${context.currentLevel}`;
	if (context.currentLevel <= context.previousLevel + 1) {
		return base;
	}

	const firstMissing = context.previousLevel + 1;
	const lastMissing = context.currentLevel - 1;
	if (firstMissing === lastMissing) {
		return `${base} (missing H${firstMissing})`;
	}

	return `${base} (missing H${firstMissing}-H${lastMissing})`;
};
