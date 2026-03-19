const env = import.meta.env as Record<string, string | undefined>;

export const SITE: {
	name: string;
	siteTitle: string;
	siteUrl: string;
	githubUrl: string;
	tagline: string;
} = {
	name: "StageFlow",
	siteTitle:
		env.VITE_SITE_TITLE ??
		"StageFlow | Podman-Native Web Accessibility QA Scanner",
	siteUrl: env.VITE_SITE_URL ?? "https://stageflow.org",
	// NOTE: Keep this pointing to a publicly accessible URL while the repository is private,
	// otherwise link-checker reports a broken-link finding on the production homepage.
	githubUrl: env.VITE_GITHUB_URL ?? "https://github.com/mattboback",
	tagline:
		env.VITE_TAGLINE ??
		"Podman-native platform for accessibility, SEO, performance, and security web scanning.",
};

export function buildSiteUrl(path = "/"): string {
	const base = SITE.siteUrl.endsWith("/")
		? SITE.siteUrl.slice(0, -1)
		: SITE.siteUrl;
	if (path === "/") {
		return base;
	}

	const suffix = path.startsWith("/") ? path : `/${path}`;
	return `${base}${suffix}`;
}
