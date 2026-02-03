const env = import.meta.env as Record<string, string | undefined>;

export const SITE: {
	name: string;
	siteTitle: string;
	siteUrl: string;
	githubUrl: string;
	tagline: string;
} = {
	name: 'StageFlow',
	siteTitle: env.VITE_SITE_TITLE ?? 'StageFlow',
	siteUrl: env.VITE_SITE_URL ?? 'https://stageflow.org',
	githubUrl: env.VITE_GITHUB_URL ?? 'https://github.com/mattboback/stageflow',
	tagline: env.VITE_TAGLINE ?? 'Podman-native web accessibility scanning platform'
};

export function buildSiteUrl(path = '/'): string {
	const base = SITE.siteUrl.endsWith('/') ? SITE.siteUrl.slice(0, -1) : SITE.siteUrl;
	if (path === '/') {
		return base;
	}

	const suffix = path.startsWith('/') ? path : `/${path}`;
	return `${base}${suffix}`;
}
