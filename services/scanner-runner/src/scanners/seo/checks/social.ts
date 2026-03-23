import type { SEOCheck } from '../types';

export const SOCIAL_CHECKS: SEOCheck[] = [
	{
		id: 'missing-og-tags',
		title: 'Missing Open Graph Tags',
		severity: 'moderate',
		category: 'social',
		helpUrl: 'https://ogp.me/',
		check: (data) => {
			const required = ['og:title', 'og:description', 'og:image'];
			const missing = required.filter((tag) => !data.ogTags[tag]);
			if (missing.length > 0) {
				return {
					passed: false,
					message: `Missing Open Graph tags: ${missing.join(', ')}. These improve social media sharing.`,
					details: { missing, present: Object.keys(data.ogTags) }
				};
			}

			return null;
		}
	}
];
