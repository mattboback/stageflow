import { isUnifiedReport, sanitizeProjectConfiguration } from './projects';
import type { LocalProjectConfiguration, ProjectDiff } from './projects';
import { normalizeReviewVerdicts, type ReviewVerdictsByJob } from './report/review-verdict';
import type { ScreenshotArtifact } from './types/scan';
import type { UnifiedReport } from './types/unified-report';
import { parseVerificationRequest, type VerificationRequest } from './verification';

export const PORTABLE_AUDIT_SCHEMA = 'stageflow/portable-audit@v1' as const;
export const MAX_AUDIT_BYTES = 96 * 1024 * 1024;
const MAX_IMAGE_BYTES = 8 * 1024 * 1024;
const MAX_MEDIA_BYTES = 48 * 1024 * 1024;
const IMAGE_DATA_URL =
	/^data:image\/(?:png|jpeg|webp);base64,(?=[a-z\d+/])(?:[a-z\d+/]{4})*(?:[a-z\d+/]{2}==|[a-z\d+/]{3}=)?$/i;

export interface MissingAuditArtifact {
	pageId: string;
	artifactId: string;
	reason: string;
}

export interface PortableAudit {
	verification?: VerificationRequest;
	schema: typeof PORTABLE_AUDIT_SCHEMA;
	createdAt: string;
	jobId: string;
	report: UnifiedReport;
	screenshots: ScreenshotArtifact[];
	reviews: ReviewVerdictsByJob;
	configuration?: LocalProjectConfiguration;
	baseline?: UnifiedReport;
	comparison?: ProjectDiff;
	missingArtifacts: MissingAuditArtifact[];
}

export interface OfflineViewerAssets {
	javascript: string;
	css: string;
}

export interface PortableAuditInput {
	verification?: VerificationRequest;
	jobId: string;
	report: UnifiedReport;
	screenshots: ScreenshotArtifact[];
	reviews?: ReviewVerdictsByJob;
	configuration?: LocalProjectConfiguration;
	baseline?: UnifiedReport;
	comparison?: ProjectDiff;
}

/** Escape the HTML parser's script terminator, including untrusted scanner text. */
export function embedAuditJSON(value: unknown): string {
	return JSON.stringify(value)
		.replace(/</g, '\\u003c')
		.replace(/>/g, '\\u003e')
		.replace(/&/g, '\\u0026')
		.replace(/\u2028/g, '\\u2028')
		.replace(/\u2029/g, '\\u2029');
}

export function renderPortableAuditHTML(audit: PortableAudit, assets: OfflineViewerAssets): string {
	const normalized = parsePortableAudit(audit);
	if (!normalized) throw new Error('This audit does not contain valid portable data.');
	const data = embedAuditJSON(normalized);
	if (new Blob([data]).size > MAX_AUDIT_BYTES)
		throw new Error('This audit exceeds the 96 MiB export limit.');
	const javascript = assets.javascript.replace(/<\/script/gi, '<\\/script');
	const css = assets.css.replace(/<\/style/gi, '<\\/style');
	return `<!doctype html><html lang="en" data-theme="light"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src 'none'; base-uri 'none'; form-action 'none'"><title>StageFlow offline audit</title><style id="stageflow-offline-css">${css}</style></head><body><a class="skip-link" href="#main">Skip to main content</a><div id="stageflow-offline-root"></div><script id="stageflow-audit" type="application/json">${data}</script><script id="stageflow-offline-js">${javascript}</script></body></html>`;
}

/** Actual image bytes anchor screenshot-derived contrast review decisions. */
export async function portableScreenshotDigests(
	screenshots: ScreenshotArtifact[]
): Promise<Record<string, string>> {
	const digests: Record<string, string> = {};
	if (!globalThis.crypto?.subtle) return digests;
	for (const screenshot of screenshots) {
		if (!IMAGE_DATA_URL.test(screenshot.url)) continue;
		const encoded = screenshot.url.slice(screenshot.url.indexOf(',') + 1);
		const decoded = atob(encoded);
		const bytes = Uint8Array.from(decoded, (character) => character.charCodeAt(0));
		const hash = await crypto.subtle.digest('SHA-256', bytes);
		digests[`${screenshot.scanner_id}|${screenshot.page_id}`] =
			`sha256:${Array.from(new Uint8Array(hash), (byte) => byte.toString(16).padStart(2, '0')).join('')}`;
	}
	return digests;
}

export function parsePortableAudit(value: unknown): PortableAudit | null {
	if (!value || typeof value !== 'object') return null;
	const candidate = value as Partial<PortableAudit>;
	if (
		candidate.schema !== PORTABLE_AUDIT_SCHEMA ||
		typeof candidate.jobId !== 'string' ||
		typeof candidate.createdAt !== 'string' ||
		!isUnifiedReport(candidate.report) ||
		!candidate.report.meta ||
		typeof candidate.report.meta.jobId !== 'string' ||
		candidate.jobId !== candidate.report.meta.jobId ||
		!Array.isArray(candidate.report.pages) ||
		!Array.isArray(candidate.report.scanners) ||
		!Array.isArray(candidate.screenshots) ||
		!Array.isArray(candidate.missingArtifacts)
	)
		return null;
	if (
		candidate.configuration &&
		(!Array.isArray(candidate.configuration.urls) ||
			!candidate.configuration.urls.every((url) => typeof url === 'string') ||
			!Array.isArray(candidate.configuration.scanners))
	)
		return null;
	if (
		candidate.comparison &&
		(candidate.comparison.schema !== 'stageflow/diff@v1' ||
			!candidate.comparison.baseline ||
			!candidate.comparison.current ||
			!candidate.comparison.delta ||
			!Array.isArray(candidate.comparison.new) ||
			!Array.isArray(candidate.comparison.fixed))
	)
		return null;
	const screenshots: ScreenshotArtifact[] = [];
	let mediaSize = 0;
	for (const screenshot of candidate.screenshots) {
		if (
			!screenshot ||
			screenshot.kind !== 'page_overview' ||
			typeof screenshot.artifact_id !== 'string' ||
			typeof screenshot.scanner_id !== 'string' ||
			typeof screenshot.page_id !== 'string' ||
			typeof screenshot.url !== 'string' ||
			!IMAGE_DATA_URL.test(screenshot.url) ||
			screenshot.url.length > MAX_IMAGE_BYTES * 1.4
		)
			return null;
		mediaSize += screenshot.url.length;
		if (mediaSize > MAX_MEDIA_BYTES * 1.4 || screenshots.length >= 100) return null;
		screenshots.push(copyScreenshot(screenshot, screenshot.url));
	}
	const missingArtifacts: MissingAuditArtifact[] = [];
	for (const missing of candidate.missingArtifacts) {
		if (
			!missing ||
			typeof missing.pageId !== 'string' ||
			typeof missing.artifactId !== 'string' ||
			typeof missing.reason !== 'string'
		)
			return null;
		missingArtifacts.push(missing);
	}
	const verification = parseVerificationRequest(candidate.verification);
	if (verification && verification.jobId !== candidate.jobId) return null;
	return {
		...(verification ? { verification } : {}),
		schema: PORTABLE_AUDIT_SCHEMA,
		jobId: candidate.jobId,
		createdAt: candidate.createdAt,
		report: candidate.report,
		screenshots,
		missingArtifacts,
		reviews: scopedReviews(candidate.reviews, candidate.jobId, candidate.baseline?.meta?.jobId),
		...(candidate.configuration
			? { configuration: sanitizeProjectConfiguration(candidate.configuration) }
			: {}),
		...(isUnifiedReport(candidate.baseline) ? { baseline: candidate.baseline } : {}),
		...(candidate.comparison ? { comparison: candidate.comparison } : {})
	};
}

export function portableScanEnvelope(report: UnifiedReport) {
	return {
		schema: 'stageflow-cli/report@v1',
		cli: { version: 'portable-audit' },
		api: { base_url: '' },
		job: { id: report.meta.jobId, state: 'complete' },
		links: { job: '', results: '' },
		urls: report.pages.map((page) => page.url),
		filters: {
			max_issues: 0,
			issues_returned: report.issues.length,
			issues_total: report.issues.length,
			truncated: false,
			sort: 'severity'
		},
		report
	};
}

export function downloadAuditFile(name: string, value: string, mime: string): void {
	const url = URL.createObjectURL(new Blob([value], { type: mime }));
	const anchor = document.createElement('a');
	anchor.href = url;
	anchor.download = name;
	anchor.click();
	setTimeout(() => URL.revokeObjectURL(url), 1000);
}

async function boundedBytes(response: Response, maximum: number): Promise<Uint8Array> {
	if (!response.ok) throw new Error(`Download returned HTTP ${response.status}.`);
	if (Number(response.headers.get('content-length')) > maximum)
		throw new Error('Artifact exceeds the size limit.');
	if (!response.body) throw new Error('Artifact download has no body.');
	const reader = response.body.getReader();
	const chunks: Uint8Array[] = [];
	let length = 0;
	try {
		while (true) {
			const chunk = await reader.read();
			if (chunk.done) break;
			length += chunk.value.byteLength;
			if (length > maximum) throw new Error('Artifact exceeds the size limit.');
			chunks.push(chunk.value);
		}
	} catch (error) {
		await reader.cancel().catch(() => undefined);
		throw error;
	} finally {
		reader.releaseLock();
	}
	const result = new Uint8Array(length);
	let offset = 0;
	for (const chunk of chunks) {
		result.set(chunk, offset);
		offset += chunk.byteLength;
	}
	return result;
}

function scopedReviews(value: unknown, jobId: string, baselineJobId?: string): ReviewVerdictsByJob {
	const reviews = normalizeReviewVerdicts(value);
	return Object.fromEntries(
		Object.entries(reviews).filter(([id]) => id === jobId || id === baselineJobId)
	);
}

function copyScreenshot(screenshot: ScreenshotArtifact, url: string): ScreenshotArtifact {
	return {
		kind: 'page_overview',
		artifact_id: screenshot.artifact_id,
		scanner_id: screenshot.scanner_id,
		page_id: screenshot.page_id,
		url,
		...(typeof screenshot.page_url === 'string' ? { page_url: screenshot.page_url } : {}),
		...(typeof screenshot.captured_at === 'string' ? { captured_at: screenshot.captured_at } : {})
	};
}

function imageDataURL(bytes: Uint8Array, mime: string): string {
	const slices: string[] = [];
	for (let offset = 0; offset < bytes.length; offset += 8192) {
		slices.push(String.fromCharCode(...bytes.subarray(offset, offset + 8192)));
	}
	return `data:${mime};base64,${btoa(slices.join(''))}`;
}

export async function preparePortableAudit(input: PortableAuditInput): Promise<PortableAudit> {
	const screenshots: ScreenshotArtifact[] = [];
	const missingArtifacts: MissingAuditArtifact[] = [];
	let total = 0;
	const deadline = AbortSignal.timeout(60_000);
	for (const screenshot of input.screenshots) {
		try {
			if (screenshots.length >= 100) throw new Error('The 100-screenshot limit was reached.');
			let dataURL: string;
			if (IMAGE_DATA_URL.test(screenshot.url)) {
				if (screenshot.url.length > MAX_IMAGE_BYTES * 1.4)
					throw new Error('Artifact exceeds the size limit.');
				dataURL = screenshot.url;
			} else {
				const target = new URL(screenshot.url, window.location.href);
				if (!['http:', 'https:'].includes(target.protocol))
					throw new Error('Unsupported image URL.');
				const response = await fetch(target, {
					credentials: 'omit',
					signal: AbortSignal.any([deadline, AbortSignal.timeout(15_000)])
				});
				if (!response.ok) throw new Error(`Download returned HTTP ${response.status}.`);
				const mime = response.headers.get('content-type')?.split(';')[0]?.trim() ?? '';
				if (!['image/png', 'image/jpeg', 'image/webp'].includes(mime))
					throw new Error('Artifact is not a supported screenshot.');
				dataURL = imageDataURL(await boundedBytes(response, MAX_IMAGE_BYTES), mime);
			}
			if (total + dataURL.length > MAX_MEDIA_BYTES * 1.4)
				throw new Error('The 48 MiB screenshot budget was reached.');
			total += dataURL.length;
			screenshots.push(copyScreenshot(screenshot, dataURL));
		} catch (error) {
			missingArtifacts.push({
				pageId: screenshot.page_id,
				artifactId: screenshot.artifact_id,
				reason: error instanceof Error ? error.message : 'Screenshot could not be downloaded.'
			});
		}
	}
	for (const page of input.report.pages) {
		if (!input.screenshots.some((screenshot) => screenshot.page_id === page.id)) {
			missingArtifacts.push({
				pageId: page.id,
				artifactId: '',
				reason: 'No captured screenshot is available for this page.'
			});
		}
	}
	const verification = parseVerificationRequest(input.verification);
	return {
		schema: PORTABLE_AUDIT_SCHEMA,
		createdAt: new Date().toISOString(),
		jobId: input.jobId,
		report: input.report,
		screenshots,
		missingArtifacts,
		reviews: scopedReviews(input.reviews, input.jobId, input.baseline?.meta.jobId),
		...(input.configuration
			? { configuration: sanitizeProjectConfiguration(input.configuration) }
			: {}),
		...(input.baseline ? { baseline: input.baseline } : {}),
		...(input.comparison ? { comparison: input.comparison } : {}),
		...(verification?.jobId === input.jobId ? { verification } : {})
	};
}

export async function loadOfflineViewerAssets(): Promise<OfflineViewerAssets> {
	const signal = AbortSignal.timeout(15_000);
	const load = async (name: string, max: number) =>
		new TextDecoder().decode(
			await boundedBytes(await fetch(`/assets/offline-viewer.${name}`, { signal }), max)
		);
	const [javascript, css] = await Promise.all([
		load('js', 8 * 1024 * 1024),
		load('css', 1024 * 1024)
	]);
	if (!javascript.includes('stageflow-offline-root') || !css.trim())
		throw new Error('The offline viewer is unavailable. Build or reload StageFlow and try again.');
	return { javascript, css };
}
