import type { UnifiedReport } from './types/unified-report';
import type { LocalProjectConfiguration } from './projects';

type Guard = (value: unknown) => boolean;
const string: Guard = (value) => typeof value === 'string';
const number: Guard = (value) => typeof value === 'number' && Number.isFinite(value);
const boolean: Guard = (value) => typeof value === 'boolean';
const record: Guard = (value) =>
	value !== null && typeof value === 'object' && !Array.isArray(value);
const array =
	(guard: Guard): Guard =>
	(value) =>
		Array.isArray(value) && value.every(guard);
const oneOf =
	(...values: string[]): Guard =>
	(value) =>
		typeof value === 'string' && values.includes(value);
const dictionary =
	(guard: Guard): Guard =>
	(value) =>
		record(value) && Object.values(value as Record<string, unknown>).every(guard);

/** Required contract fields must exist; known optional fields are checked when supplied. */
function object(required: Record<string, Guard>, optional: Record<string, Guard> = {}): Guard {
	return (value) => {
		if (!record(value)) return false;
		const candidate = value as Record<string, unknown>;
		return (
			Object.entries(required).every(([key, guard]) => guard(candidate[key])) &&
			Object.entries(optional).every(
				([key, guard]) => candidate[key] === undefined || guard(candidate[key])
			)
		);
	};
}
const strings = (...keys: string[]) => Object.fromEntries(keys.map((key) => [key, string]));
const numbers = (...keys: string[]) => Object.fromEntries(keys.map((key) => [key, number]));
const severity = oneOf('critical', 'serious', 'moderate', 'minor', 'info');
const profile = oneOf('production', 'staging');
const severityCounts = object(numbers('critical', 'serious', 'moderate', 'minor'), {
	info: number
});
const boundingBox = object(numbers('x', 'y', 'width', 'height'));
const occurrence = object(
	{},
	{
		...strings(
			'selector',
			'html',
			'contextHtml',
			'ancestorPath',
			'failureSummary',
			'textSnippet',
			'elementId',
			'pageId',
			'label'
		),
		target: array(string),
		artifactIds: array(string),
		boundingBox
	}
);
const issue = object(
	{
		...strings('id', 'scanner', 'ruleId', 'title', 'description', 'pageId', 'pageUrl'),
		severity,
		elementCount: number
	},
	{
		...strings('severityRaw', 'helpUrl', 'category', 'howToFix'),
		wcagTags: array(string),
		elementsTruncated: boolean,
		occurrences: array(occurrence),
		scannerData: record,
		friendlyNode: object(
			{},
			strings('label', 'tagName', 'role', 'name', 'region', 'textSnippet', 'selector')
		),
		locationInfo: object({}, numbers('scrollY', 'viewportHeight', 'docHeight', 'position')),
		userImpact: object(
			{},
			{
				...strings('statement', 'userStory'),
				affectedGroups: array(
					oneOf('blind', 'low-vision', 'motor', 'cognitive', 'deaf', 'vestibular', 'all')
				),
				severity: oneOf('blocking', 'degraded', 'inconvenient')
			}
		)
	}
);
const scanContext = object(
	{
		expectedPages: array(string),
		identityScope: oneOf('path', 'origin'),
		scannerConfigFingerprints: dictionary(string),
		purpose: oneOf('full', 'verification'),
		authenticated: boolean,
		inputType: oneOf('urls', 'zip')
	},
	{
		viewport: object(numbers('width', 'height')),
		scannerConfigurations: dictionary(record),
		pageConfigFingerprints: dictionary(string),
		verificationEligiblePages: dictionary(boolean)
	}
);
const pageOverview = object({
	screenshotFilename: string,
	...numbers('pageWidth', 'pageHeight'),
	elements: array(
		object({
			...strings('issueId', 'ruleId', 'selector'),
			severity,
			...numbers(
				'nodeIndex',
				'xPercent',
				'yPercent',
				'widthPercent',
				'heightPercent',
				'x',
				'y',
				'width',
				'height'
			)
		})
	)
});
const page = object(
	{ ...strings('id', 'url'), ...numbers('issueCount', 'durationMs') },
	{
		...strings('path', 'startedAt', 'finishedAt'),
		bySeverity: severityCounts,
		pageOverview
	}
);
const scanner = object(
	{ id: string, status: oneOf('success', 'partial', 'failed', 'skipped') },
	{
		...strings(
			'name',
			'error',
			'toolVersion',
			'startedAt',
			'completedAt',
			'resultsPath',
			'reportPath'
		),
		...numbers('issueCount', 'durationMs'),
		severity: severityCounts,
		pages: array(
			object(
				{ pageId: string, status: oneOf('success', 'failed', 'skipped') },
				{
					finalUrl: string,
					rules: array(
						object(
							{
								ruleId: string,
								outcome: oneOf('pass', 'fail', 'incomplete', 'inapplicable', 'error')
							},
							{ subjectIds: array(string) }
						)
					)
				}
			)
		)
	}
);
const report = object(
	{
		version: (value) => string(value) && (value as string).startsWith('2.'),
		meta: object(
			{ jobId: string },
			{
				...strings('baseUrl', 'scannedAt', 'completedAt', 'policyVersion'),
				durationMs: number,
				profile,
				scanContext
			}
		),
		summary: object(
			{ ...numbers('totalIssues', 'pagesScanned', 'pagesWithIssues'), bySeverity: severityCounts },
			{
				score: number,
				scoreGrade: string,
				byScanner: dictionary(number),
				lighthouseCategories: array(
					object({
						id: oneOf('performance', 'accessibility', 'best-practices', 'seo', 'pwa'),
						title: string,
						avgScore: number
					})
				)
			}
		),
		scanners: array(scanner),
		pages: array(page),
		issues: array(issue)
	},
	{
		artifacts: array(object(strings('id', 'type'), strings('path', 'mime', 'dataUri'))),
		errors: array(
			object(
				{
					scope: oneOf('scanner', 'page', 'global'),
					...strings('code', 'message'),
					retryable: boolean
				},
				strings('scannerId', 'pageId')
			)
		),
		manualChecks: array(
			object(
				{ ...strings('scanner', 'ruleId', 'title'), pageCount: number },
				{ ...strings('description', 'helpUrl'), pageIds: array(string) }
			)
		),
		assessment: object(
			{
				profile,
				policyVersion: string,
				...numbers('actionable', 'expected', 'needsReview'),
				dispositionByIssue: dictionary(oneOf('actionable', 'expected', 'needs-review'))
			},
			{ score: number, scoreGrade: string }
		)
	}
);

/** Structural report validation, compatible with older 2.x files and future extra fields. */
export function isUnifiedReport(value: unknown): value is UnifiedReport {
	return report(value);
}

/** Validate before sanitizing; legacy browser/highlight values still normalize on import. */
export function isProjectConfiguration(value: unknown): value is LocalProjectConfiguration {
	return object(
		{
			urls: array(string),
			scanners: array(object({ id: string, enabled: boolean }, { config: record }))
		},
		{
			browser: string,
			highlightStyle: string,
			profile
		}
	)(value);
}

/** Comparison findings are rendered like report findings and need the same boundary. */
export function isProjectDiff(value: unknown): value is import('./projects').ProjectDiff {
	const side = object({ jobId: string, totalIssues: number }, { score: number });
	return object(
		{
			schema: oneOf('stageflow/diff@v1'),
			baseline: side,
			current: side,
			delta: object(numbers('newIssues', 'fixedIssues', 'unchangedIssues'), { scoreDelta: number }),
			new: array(issue),
			fixed: array(issue)
		},
		{
			skippedScanners: array(string),
			incompatibleIdentity: boolean,
			notComparedPages: array(string),
			contextLimited: boolean
		}
	)(value);
}
