import type {
	IssueDetail,
	IssueSeverity,
	PageOverviewElement,
	ScannerSummary,
	SeverityCounts,
	UnifiedReport
} from '../types/unified-report';

const OCCURRENCE_ID_DELIMITER = '--occ-';
type IssueOccurrence = NonNullable<IssueDetail['occurrences']>[number];

function createEmptySeverityCounts(): SeverityCounts {
	return {
		critical: 0,
		serious: 0,
		moderate: 0,
		minor: 0,
		info: 0
	};
}

function incrementSeverity(counts: SeverityCounts, severity: IssueSeverity): void {
	counts[severity] += 1;
}

function cloneSeverityCounts(counts?: SeverityCounts): SeverityCounts {
	if (!counts) {
		return createEmptySeverityCounts();
	}

	return {
		critical: counts.critical,
		serious: counts.serious,
		moderate: counts.moderate,
		minor: counts.minor,
		info: counts.info ?? 0
	};
}

function asOptionalString(value: unknown): string | undefined {
	return typeof value === 'string' ? value : undefined;
}

function asOptionalStringArray(value: unknown): string[] | undefined {
	if (!Array.isArray(value)) {
		return undefined;
	}

	const stringValues = value.filter((entry): entry is string => typeof entry === 'string');
	return stringValues.length > 0 ? stringValues : undefined;
}

function asOptionalBoundingBox(value: unknown): IssueOccurrence['boundingBox'] {
	if (typeof value !== 'object' || value === null) {
		return undefined;
	}

	const maybe = value as Record<string, unknown>;
	if (
		typeof maybe.x === 'number' &&
		typeof maybe.y === 'number' &&
		typeof maybe.width === 'number' &&
		typeof maybe.height === 'number'
	) {
		return {
			x: maybe.x,
			y: maybe.y,
			width: maybe.width,
			height: maybe.height
		};
	}

	return undefined;
}

function normalizeOccurrence(
	occurrence: unknown,
	derivedIssueId: string,
	pageId: string
): IssueOccurrence {
	const source =
		typeof occurrence === 'object' && occurrence !== null
			? (occurrence as Record<string, unknown>)
			: {};

	const target = asOptionalStringArray(source.target);
	const selector = asOptionalString(source.selector);
	const html = asOptionalString(source.html);
	const contextHtml = asOptionalString(source.contextHtml);
	const ancestorPath = asOptionalString(source.ancestorPath);
	const failureSummary = asOptionalString(source.failureSummary);
	const textSnippet = asOptionalString(source.textSnippet);
	const boundingBox = asOptionalBoundingBox(source.boundingBox);
	const label = asOptionalString(source.label);
	const artifactIds = asOptionalStringArray(source.artifactIds);
	const elementPageId = asOptionalString(source.pageId) ?? pageId;
	return {
		...(target !== undefined ? { target } : {}),
		...(selector !== undefined ? { selector } : {}),
		...(html !== undefined ? { html } : {}),
		...(contextHtml !== undefined ? { contextHtml } : {}),
		...(ancestorPath !== undefined ? { ancestorPath } : {}),
		...(failureSummary !== undefined ? { failureSummary } : {}),
		...(textSnippet !== undefined ? { textSnippet } : {}),
		...(boundingBox !== undefined ? { boundingBox } : {}),
		...(label !== undefined ? { label } : {}),
		...(artifactIds !== undefined ? { artifactIds } : {}),
		elementId: `${derivedIssueId}-el-0`,
		pageId: elementPageId
	};
}

function buildDerivedIssueId(baseIssueId: string, occurrenceIndex: number): string {
	if (occurrenceIndex === 0) {
		return baseIssueId;
	}

	return `${baseIssueId}${OCCURRENCE_ID_DELIMITER}${occurrenceIndex + 1}`;
}

interface ExpandedIssues {
	issues: IssueDetail[];
	derivedIdsByIssueId: Map<string, string[]>;
}

function expandIssuesByOccurrence(issues: IssueDetail[]): ExpandedIssues {
	const expanded: IssueDetail[] = [];
	const derivedIdsByIssueId = new Map<string, string[]>();

	for (const issue of issues) {
		const occurrences = Array.isArray(issue.occurrences) ? issue.occurrences : [];
		if (occurrences.length === 0) {
			expanded.push({ ...issue });
			derivedIdsByIssueId.set(issue.id, [issue.id]);
			continue;
		}

		const derivedIds: string[] = [];
		for (let occurrenceIndex = 0; occurrenceIndex < occurrences.length; occurrenceIndex += 1) {
			const occurrence = occurrences[occurrenceIndex];
			const derivedIssueId = buildDerivedIssueId(issue.id, occurrenceIndex);
			const normalizedOccurrence = normalizeOccurrence(occurrence, derivedIssueId, issue.pageId);

			expanded.push({
				...issue,
				id: derivedIssueId,
				elementCount: 1,
				elementsTruncated: false,
				occurrences: [normalizedOccurrence]
			});

			derivedIds.push(derivedIssueId);
		}

		derivedIdsByIssueId.set(issue.id, derivedIds);
	}

	return { issues: expanded, derivedIdsByIssueId };
}

function buildSelectorIndex(issues: IssueDetail[]): Map<string, string> {
	const index = new Map<string, string>();
	for (const issue of issues) {
		const occurrence = issue.occurrences?.[0];
		if (!occurrence) continue;
		const selector = occurrence.selector ?? occurrence.target?.[0];
		if (selector) {
			index.set(`${issue.ruleId}|${selector}`, issue.id);
		}
	}
	return index;
}

function remapPageOverviewElements(
	elements: PageOverviewElement[],
	derivedIdsByIssueId: Map<string, string[]>,
	expandedIssues: IssueDetail[]
): PageOverviewElement[] {
	const selectorIndex = buildSelectorIndex(expandedIssues);

	return elements.map((element) => {
		const derivedIssueIds = derivedIdsByIssueId.get(element.issueId);
		if (!derivedIssueIds || derivedIssueIds.length === 0) {
			const fallbackId = selectorIndex.get(`${element.ruleId}|${element.selector}`);
			if (fallbackId) {
				return { ...element, issueId: fallbackId, nodeIndex: 0 };
			}
			return element;
		}

		const remappedIssueId = derivedIssueIds[element.nodeIndex];
		if (remappedIssueId !== undefined) {
			return { ...element, issueId: remappedIssueId, nodeIndex: 0 };
		}

		const selectorFallbackId = selectorIndex.get(`${element.ruleId}|${element.selector}`);
		if (selectorFallbackId) {
			return { ...element, issueId: selectorFallbackId, nodeIndex: 0 };
		}

		return { ...element, issueId: derivedIssueIds[0], nodeIndex: 0 };
	});
}

function buildScannerSummaries(
	scanners: ScannerSummary[],
	issues: IssueDetail[]
): { scanners: ScannerSummary[]; byScanner: Record<string, number> } {
	const severityByScanner = new Map<string, SeverityCounts>();
	const countByScanner = new Map<string, number>();

	for (const issue of issues) {
		const currentCount = countByScanner.get(issue.scanner) ?? 0;
		countByScanner.set(issue.scanner, currentCount + 1);

		const currentSeverity = severityByScanner.get(issue.scanner) ?? createEmptySeverityCounts();
		incrementSeverity(currentSeverity, issue.severity);
		severityByScanner.set(issue.scanner, currentSeverity);
	}

	const nextScanners = scanners.map((scanner) => ({
		...scanner,
		issueCount: countByScanner.get(scanner.id) ?? 0,
		severity: cloneSeverityCounts(severityByScanner.get(scanner.id))
	}));

	const byScanner: Record<string, number> = {};
	for (const scanner of nextScanners) {
		byScanner[scanner.id] = scanner.issueCount;
	}
	for (const [scannerId, issueCount] of countByScanner.entries()) {
		byScanner[scannerId] = issueCount;
	}

	return { scanners: nextScanners, byScanner };
}

export function buildOccurrenceModeReport(report: UnifiedReport): UnifiedReport {
	const { issues: occurrenceIssues, derivedIdsByIssueId } = expandIssuesByOccurrence(report.issues);

	const issuesByPageId = new Map<string, IssueDetail[]>();
	const summaryBySeverity = createEmptySeverityCounts();
	for (const issue of occurrenceIssues) {
		const pageIssues = issuesByPageId.get(issue.pageId) ?? [];
		pageIssues.push(issue);
		issuesByPageId.set(issue.pageId, pageIssues);

		incrementSeverity(summaryBySeverity, issue.severity);
	}

	const pages = report.pages.map((page) => {
		const pageIssues = issuesByPageId.get(page.id) ?? [];
		const pageBySeverity = createEmptySeverityCounts();
		for (const issue of pageIssues) {
			incrementSeverity(pageBySeverity, issue.severity);
		}

		const updatedPageOverview = page.pageOverview
			? {
					...page.pageOverview,
					elements: remapPageOverviewElements(
						page.pageOverview.elements,
						derivedIdsByIssueId,
						occurrenceIssues
					)
				}
			: undefined;
		return {
			...page,
			issueCount: pageIssues.length,
			bySeverity: pageBySeverity,
			...(updatedPageOverview !== undefined ? { pageOverview: updatedPageOverview } : {})
		};
	});

	const { scanners, byScanner } = buildScannerSummaries(report.scanners, occurrenceIssues);
	const pagesWithIssues = pages.filter((page) => page.issueCount > 0).length;

	return {
		...report,
		summary: {
			...report.summary,
			totalIssues: occurrenceIssues.length,
			bySeverity: summaryBySeverity,
			byScanner,
			pagesScanned: pages.length,
			pagesWithIssues
		},
		scanners,
		pages,
		issues: occurrenceIssues
	};
}
