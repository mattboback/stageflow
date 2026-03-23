import type { SEOCheck } from '../types';

export const META_CHECKS: SEOCheck[] = [
	{
		id: 'missing-title',
		title: 'Missing Page Title',
		severity: 'critical',
		category: 'meta',
		helpUrl: 'https://moz.com/learn/seo/title-tag',
		check: (data) => {
			if (!data.title) {
				return {
					passed: false,
					message:
						'Page is missing a title tag. Title tags are crucial for SEO and user experience.'
				};
			}

			return null;
		}
	},
	{
		id: 'title-length',
		title: 'Title Tag Length',
		severity: 'moderate',
		category: 'meta',
		helpUrl: 'https://moz.com/learn/seo/title-tag',
		check: (data) => {
			if (data.title) {
				const len = data.title.length;
				if (len < 30) {
					return {
						passed: false,
						message: `Title is too short (${len} characters). Recommended: 50-60 characters.`,
						details: { length: len, recommended: '50-60' }
					};
				}

				if (len > 60) {
					return {
						passed: false,
						message: `Title is too long (${len} characters). It may be truncated in search results. Recommended: 50-60 characters.`,
						details: { length: len, recommended: '50-60' }
					};
				}
			}

			return null;
		}
	},
	{
		id: 'missing-description',
		title: 'Missing Meta Description',
		severity: 'serious',
		category: 'meta',
		helpUrl: 'https://moz.com/learn/seo/meta-description',
		check: (data) => {
			if (!data.description) {
				return {
					passed: false,
					message:
						'Page is missing a meta description. This affects click-through rates from search results.'
				};
			}

			return null;
		}
	},
	{
		id: 'description-length',
		title: 'Meta Description Length',
		severity: 'moderate',
		category: 'meta',
		helpUrl: 'https://moz.com/learn/seo/meta-description',
		check: (data) => {
			if (data.description) {
				const len = data.description.length;
				if (len < 70) {
					return {
						passed: false,
						message: `Meta description is too short (${len} characters). Recommended: 120-160 characters.`,
						details: { length: len, recommended: '120-160' }
					};
				}

				if (len > 160) {
					return {
						passed: false,
						message: `Meta description is too long (${len} characters). It may be truncated. Recommended: 120-160 characters.`,
						details: { length: len, recommended: '120-160' }
					};
				}
			}

			return null;
		}
	}
];
