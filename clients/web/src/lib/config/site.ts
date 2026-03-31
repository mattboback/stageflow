const env = import.meta.env as Record<string, string | undefined>;

const DEFAULT_GITHUB_URL = 'https://github.com/mattboback';
const DEFAULT_TAGLINE =
	'Podman-native platform for accessibility, SEO, performance, and security web scanning';

function normalizeGithubUrl(value: string | undefined): string {
	if (!value) {
		return DEFAULT_GITHUB_URL;
	}

	try {
		const url = new URL(value);
		if (url.hostname === 'github.com') {
			const pathname = url.pathname.replace(/\/+$/, '');
			if (pathname === '' || pathname === '/mattboback' || pathname === '/mattboback/stageflow') {
				return DEFAULT_GITHUB_URL;
			}
		}

		return url.toString();
	} catch {
		return DEFAULT_GITHUB_URL;
	}
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
	siteTitle: env.VITE_SITE_TITLE ?? 'StageFlow | Podman-Native Web Accessibility QA Scanner',
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
