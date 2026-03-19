export type WcagConformanceLevel = 'A' | 'AA' | 'AAA' | null;

/**
 * Normalize various wcag tag formats to a criterion number (e.g. "1.4.3").
 */
export function normalizeWcagTag(tag: string): string | null {
	const normalized = tag.toLowerCase().trim();

	// Match patterns like "wcag111", "wcag412"
	const noDotMatch = /^wcag(\d)(\d)(\d+)$/.exec(normalized);
	if (noDotMatch) {
		return `${noDotMatch[1]}.${noDotMatch[2]}.${noDotMatch[3]}`;
	}

	// Match patterns like "wcag1.1.1", "wcag2.4.4"
	const dotMatch = /^wcag(\d)\.(\d)\.(\d+)$/.exec(normalized);
	if (dotMatch) {
		return `${dotMatch[1]}.${dotMatch[2]}.${dotMatch[3]}`;
	}

	// Some tags are already just the criterion (e.g. "1.4.3")
	const plainMatch = /^(\d)\.(\d)\.(\d+)$/.exec(normalized);
	if (plainMatch) {
		return `${plainMatch[1]}.${plainMatch[2]}.${plainMatch[3]}`;
	}

	return null;
}

export function getWcagUnderstandingUrl(criterion: string): string {
	return `https://www.w3.org/WAI/WCAG22/Understanding/${criterion}.html`;
}

export function extractWcagCriteria(
	tags: string[] | undefined,
	wcagRef: string | undefined
): string[] {
	const criteria: string[] = [];
	const addCriterion = (criterion: string) => {
		if (criterion && !criteria.includes(criterion)) {
			criteria.push(criterion);
		}
	};

	for (const tag of tags ?? []) {
		const normalized = tag.toLowerCase().trim();

		// Match patterns like "wcag111", "wcag412"
		const noDotMatch = /^wcag(\d)(\d)(\d+)$/.exec(normalized);
		if (noDotMatch) {
			addCriterion(`${noDotMatch[1]}.${noDotMatch[2]}.${noDotMatch[3]}`);
			continue;
		}

		// Match patterns like "wcag1.1.1", "wcag2.4.4"
		const dotMatch = /^wcag(\d)\.(\d)\.(\d+)$/.exec(normalized);
		if (dotMatch) {
			addCriterion(`${dotMatch[1]}.${dotMatch[2]}.${dotMatch[3]}`);
		}
	}

	if (wcagRef) {
		const wcagRefMatch = /WCAG\s+(\d\.\d\.\d+)/i.exec(wcagRef);
		if (wcagRefMatch?.[1]) {
			addCriterion(wcagRefMatch[1]);
		}
	}

	return criteria;
}

function parseLevelFromWcagRef(wcagRef: string): WcagConformanceLevel {
	const match = wcagRef.match(/\blevel\s*(AAA|AA|A)\b/i);
	if (!match) return null;
	const normalized = match[1].toUpperCase();
	if (normalized === 'A' || normalized === 'AA' || normalized === 'AAA') {
		return normalized;
	}
	return null;
}

function parseLevelFromTags(tags: string[]): WcagConformanceLevel {
	const tagSet = new Set(tags.map((tag) => tag.toLowerCase()));

	if (tagSet.has('wcag2aaa') || tagSet.has('wcag21aaa') || tagSet.has('wcag22aaa')) return 'AAA';
	if (tagSet.has('wcag2aa') || tagSet.has('wcag21aa') || tagSet.has('wcag22aa')) return 'AA';
	if (tagSet.has('wcag2a') || tagSet.has('wcag21a') || tagSet.has('wcag22a')) return 'A';

	return null;
}

export function getIssueWcagConformanceLevel(
	tags: string[] | undefined,
	wcagRef: string | undefined
): WcagConformanceLevel {
	const fromTags = parseLevelFromTags(tags ?? []);
	if (fromTags) return fromTags;
	if (!wcagRef) return null;
	return parseLevelFromWcagRef(wcagRef);
}
