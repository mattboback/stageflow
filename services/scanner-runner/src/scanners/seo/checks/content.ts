import type { SEOCheck } from '../types';

const THIN_CONTENT_WORDS = 100;

export const CONTENT_CHECKS: SEOCheck[] = [
	{
		id: 'thin-content',
		title: 'Thin Content',
		severity: 'moderate',
		category: 'content',
		// There is no word count search engines require. 300 flagged contact forms
		// and blog indexes (298 and 267 words on a real scan) that are complete as
		// they are; this only catches pages that are close to empty.
		check: (data) => {
			if (data.wordCount < THIN_CONTENT_WORDS) {
				return {
					passed: false,
					message: `Page has only ${data.wordCount} words. Pages with thin content may rank poorly. Consider adding more valuable content.`,
					details: { wordCount: data.wordCount, recommended: `${THIN_CONTENT_WORDS}+` }
				};
			}

			return null;
		}
	}
];
