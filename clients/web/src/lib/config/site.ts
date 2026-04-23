const env = import.meta.env as Record<string, string | undefined>;

const DEFAULT_GITHUB_URL = 'https://github.com/mattboback/stageflow';
const DEFAULT_TAGLINE =
	'StageFlow is a self-hosted, Podman-native platform that runs eight web accessibility, performance, SEO, and security scanners in one report';
const DEFAULT_SITE_TITLE = 'StageFlow — Podman-Native Accessibility Scanning Platform';

function normalizeGithubUrl(value: string | undefined): string {
	const trimmed = value?.trim();
	return trimmed || DEFAULT_GITHUB_URL;
}

function normalizeTagline(value: string | undefined): string {
	return (value ?? DEFAULT_TAGLINE).trim().replace(/[.!?]+$/u, '');
}

export const SITE: {
	name: string;
	siteTitle: string;
	siteUrl: string;
	githubUrl: string;
	tagline: string;
} = {
	name: 'StageFlow',
	siteTitle: env.VITE_SITE_TITLE ?? DEFAULT_SITE_TITLE,
	siteUrl: env.VITE_SITE_URL ?? 'https://stageflow.org',
	githubUrl: normalizeGithubUrl(env.VITE_GITHUB_URL),
	tagline: normalizeTagline(env.VITE_TAGLINE)
};

export function buildSiteUrl(path = '/'): string {
	const base = SITE.siteUrl.endsWith('/') ? SITE.siteUrl.slice(0, -1) : SITE.siteUrl;
	if (path === '/') {
		return base;
	}

	const suffix = path.startsWith('/') ? path : `/${path}`;
	return `${base}${suffix}`;
}
