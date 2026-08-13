const DEFAULT_SITE_NAME = 'StageFlow';
const DEFAULT_SITE_URL = 'https://stageflow.org';
const DEFAULT_GITHUB_URL = 'https://github.com/mattboback/stageflow';
const DEFAULT_TAGLINE = 'Self-hostable frontend quality regression scanning';

function envText(value: unknown, fallback: string): string {
	return typeof value === 'string' && value.trim() ? value.trim() : fallback;
}

function absoluteUrl(value: unknown, fallback: string): string {
	const candidate = envText(value, fallback);
	try {
		const url = new URL(candidate);
		return url.protocol === 'http:' || url.protocol === 'https:'
			? url.toString().replace(/\/$/, '')
			: fallback;
	} catch {
		return fallback;
	}
}

export const SITE_TITLE = envText(import.meta.env.VITE_SITE_TITLE, DEFAULT_SITE_NAME);
export const SITE_NAME = SITE_TITLE.split(/\s+[—-]\s+/u)[0]?.trim() || DEFAULT_SITE_NAME;
export const SITE_URL = absoluteUrl(import.meta.env.VITE_SITE_URL, DEFAULT_SITE_URL);
export const GITHUB_URL = absoluteUrl(import.meta.env.VITE_GITHUB_URL, DEFAULT_GITHUB_URL);
export const DOCS_URL = `${GITHUB_URL}/tree/main/docs`;
export const SITE_TAGLINE = envText(import.meta.env.VITE_TAGLINE, DEFAULT_TAGLINE);
// Vite injects VITE_* as untyped values, so read it as unknown and narrow rather
// than calling .trim() on an implicit any.
const rawDeploymentMode: unknown = import.meta.env.VITE_DEPLOYMENT_MODE;
export const DEPLOYMENT_MODE =
	typeof rawDeploymentMode === 'string' && rawDeploymentMode.trim() === 'hosted-demo'
		? 'hosted-demo'
		: 'self-hosted';
export const IS_HOSTED_DEMO = DEPLOYMENT_MODE === 'hosted-demo';
export const SHARE_IMAGE_URL = `${SITE_URL}/social/stageflow-og.png`;

export const HOME_TITLE =
	SITE_TITLE === SITE_NAME ? `${SITE_NAME} — Frontend quality regression scanning` : SITE_TITLE;

export function buildHomeDescription(siteName: string, tagline: string): string {
	const taglinePhrase = tagline.trim().replace(/[.!?]+$/u, '');
	return `${siteName} — ${taglinePhrase}. Run accessibility, SEO, link, header, and visual checks in one report.`;
}

export const HOME_DESCRIPTION = buildHomeDescription(SITE_NAME, SITE_TAGLINE);

export const PLAYGROUND_TITLE = `${SITE_NAME} Playground — Run frontend quality scans`;
export const PLAYGROUND_DESCRIPTION = `Run ${SITE_NAME} scans against public URLs or uploaded static sites, choose scanners, stream progress, and inspect one merged quality report.`;

export function pageTitle(title: string): string {
	return `${title} — ${SITE_NAME}`;
}

type SiteMetaInput = {
	title: string;
	description: string;
	path?: string;
};

export function buildSiteMeta({ title, description, path = '/' }: SiteMetaInput) {
	const canonicalUrl = new URL(path, SITE_URL).toString();

	return [
		{ title },
		{ name: 'description', content: description },
		{ tagName: 'link', rel: 'canonical', href: canonicalUrl },
		{ property: 'og:type', content: 'website' },
		{ property: 'og:site_name', content: SITE_NAME },
		{ property: 'og:title', content: title },
		{ property: 'og:description', content: description },
		{ property: 'og:url', content: canonicalUrl },
		{ property: 'og:image', content: SHARE_IMAGE_URL },
		{ property: 'og:image:width', content: '1200' },
		{ property: 'og:image:height', content: '630' },
		{ name: 'twitter:card', content: 'summary_large_image' },
		{ name: 'twitter:title', content: title },
		{ name: 'twitter:description', content: description },
		{ name: 'twitter:image', content: SHARE_IMAGE_URL }
	];
}
