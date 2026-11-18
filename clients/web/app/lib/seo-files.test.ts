import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

import reactRouterConfig from '../../react-router.config';

describe('SEO crawler files', () => {
	const publicDir = resolve(__dirname, '../../public');

	it('provides a valid robots.txt with disallow policies and sitemap declaration', () => {
		const robotsContent = readFileSync(resolve(publicDir, 'robots.txt'), 'utf-8');

		expect(robotsContent).toContain('User-agent: *');
		expect(robotsContent).toContain('Allow: /');
		expect(robotsContent).toContain('Disallow: /scan/');
		expect(robotsContent).toContain('Disallow: /api/');
		expect(robotsContent).toContain('Disallow: /scanner-artifacts/');
		expect(robotsContent).toContain('Disallow: /scanner-staging/');
		expect(robotsContent).toContain('Disallow: /__spa-fallback.html');
		expect(robotsContent).toContain('Sitemap: https://stageflow.org/sitemap.xml');
	});

	it('provides a valid sitemap.xml listing all canonical public pages', () => {
		const sitemapContent = readFileSync(resolve(publicDir, 'sitemap.xml'), 'utf-8');

		expect(sitemapContent).toContain('xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"');
		expect(sitemapContent).toContain('<loc>https://stageflow.org/</loc>');
		expect(sitemapContent).toContain('<loc>https://stageflow.org/projects</loc>');
		expect(sitemapContent).toContain('<loc>https://stageflow.org/playground</loc>');
		expect(sitemapContent).toContain('<loc>https://stageflow.org/demo</loc>');
		expect(sitemapContent).toContain('<loc>https://stageflow.org/privacy</loc>');

		// Dynamic or private scan routes must never be in sitemap
		expect(sitemapContent).not.toContain('/scan');
		expect(sitemapContent).not.toContain('/api');
	});

	it('configures prerendering for static public routes', () => {
		const prerender = reactRouterConfig.prerender;
		expect(prerender).toEqual(
			expect.arrayContaining(['/', '/projects', '/playground', '/privacy'])
		);
	});
});
