const env = import.meta.env as Record<string, string | undefined>;

const DEFAULT_GITHUB_URL = 'https://github.com/mattboback/stageflow';
const DEFAULT_TAGLINE =
	'StageFlow is a self-hosted, Podman-native platform that supports web accessibility, performance, SEO, and security scanners in one report';
const DEFAULT_SITE_TITLE = 'StageFlow — Podman-Native Accessibility Scanning Platform';

// Container builds pass unset build args through as empty strings, so blank
// values must fall back to defaults the same way missing ones do.
function withDefault(value: string | undefined, fallback: string): string {
	const trimmed = value?.trim();
	return trimmed || fallback;
}

export const SITE: {
	name: string;
	siteTitle: string;
	siteUrl: string;
	githubUrl: string;
	tagline: string;
} = {
	name: 'StageFlow',
	siteTitle: withDefault(env.VITE_SITE_TITLE, DEFAULT_SITE_TITLE),
	siteUrl: withDefault(env.VITE_SITE_URL, 'https://stageflow.org'),
	githubUrl: withDefault(env.VITE_GITHUB_URL, DEFAULT_GITHUB_URL),
	tagline: withDefault(env.VITE_TAGLINE, DEFAULT_TAGLINE).replace(/[.!?]+$/u, '')
};

export function buildSiteUrl(path = '/'): string {
	const base = SITE.siteUrl.endsWith('/') ? SITE.siteUrl.slice(0, -1) : SITE.siteUrl;
	if (path === '/') {
		return base;
	}

	const suffix = path.startsWith('/') ? path : `/${path}`;
	return `${base}${suffix}`;
}

export function siteDisplayPath(path = '/'): string {
	const url = new URL(buildSiteUrl(path));

	return `${url.host}${url.pathname}`;
}
