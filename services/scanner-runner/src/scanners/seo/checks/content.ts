import type { SEOCheck } from '../types';

export const CONTENT_CHECKS: SEOCheck[] = [
	{
		id: 'thin-content',
		title: 'Thin Content',
		severity: 'moderate',
		category: 'content',
		check: (data) => {
			if (data.wordCount < 300) {
				return {
					passed: false,
					message: `Page has only ${data.wordCount} words. Pages with thin content may rank poorly. Consider adding more valuable content.`,
					details: { wordCount: data.wordCount, recommended: '300+' }
				};
			}

			return null;
		}
	}
];
