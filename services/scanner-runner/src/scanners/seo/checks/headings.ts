import type { SEOCheck } from "../types";

export const HEADING_CHECKS: SEOCheck[] = [
	{
		id: "missing-h1",
		title: "Missing H1 Heading",
		severity: "serious",
		category: "headings",
		helpUrl: "https://moz.com/learn/seo/headings",
		check: (data) => {
			const h1s = data.headings.filter((h) => h.level === 1);
			if (h1s.length === 0) {
				return {
					passed: false,
					message:
						"Page is missing an H1 heading. Every page should have exactly one H1.",
				};
			}

			return null;
		},
	},
	{
		id: "multiple-h1",
		title: "Multiple H1 Headings",
		severity: "moderate",
		category: "headings",
		helpUrl: "https://moz.com/learn/seo/headings",
		check: (data) => {
			const h1s = data.headings.filter((h) => h.level === 1);
			if (h1s.length > 1) {
				return {
					passed: false,
					message: `Page has ${h1s.length} H1 headings. Best practice is to have exactly one H1.`,
					details: { h1s: h1s.map((h) => h.text) },
				};
			}

			return null;
		},
	},
	{
		id: "heading-hierarchy",
		title: "Heading Hierarchy",
		severity: "minor",
		category: "headings",
		check: (data) => {
			const levels = data.headings.map((h) => h.level);
			for (let i = 1; i < levels.length; i++) {
				const current = levels[i];
				const previous = levels[i - 1];
				if (
					current !== undefined &&
					previous !== undefined &&
					current > previous + 1
				) {
					return {
						passed: false,
						message: `Heading hierarchy skips levels (H${previous} followed by H${current}). This may confuse screen readers and search engines.`,
						details: { headings: data.headings.slice(0, 10) },
					};
				}
			}

			return null;
		},
	},
];
