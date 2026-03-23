import type { SEOCheck } from '../types';

export const IMAGE_CHECKS: SEOCheck[] = [
	{
		id: 'images-missing-alt',
		title: 'Images Missing Alt Text',
		severity: 'serious',
		category: 'images',
		helpUrl: 'https://moz.com/learn/seo/alt-text',
		check: (data) => {
			const missingAlt = data.images.filter((img) => !img.alt);
			if (missingAlt.length > 0) {
				return {
					passed: false,
					message: `${missingAlt.length} image(s) are missing alt text. Alt text is important for accessibility and SEO.`,
					details: { images: missingAlt.slice(0, 5).map((i) => i.src) }
				};
			}

			return null;
		}
	}
];
