export const SITE_URL = 'https://stageflow.org';
export const SITE_NAME = 'StageFlow';
export const SHARE_IMAGE_URL = `${SITE_URL}/social/stageflow-og.png`;

export const HOME_TITLE = 'StageFlow — Frontend quality regression scanning';
export const HOME_DESCRIPTION =
	'StageFlow is a self-hostable frontend quality platform that runs accessibility, SEO, link, header, and visual checks in one report.';

export const PLAYGROUND_TITLE = 'StageFlow Playground — Run frontend quality scans';
export const PLAYGROUND_DESCRIPTION =
	'Run StageFlow scans against public URLs or uploaded static sites, choose scanners, stream progress, and inspect one merged quality report.';

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
