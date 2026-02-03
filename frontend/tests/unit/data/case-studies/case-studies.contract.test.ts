import { caseStudies } from '$lib/data/case-studies';
import { describe, expect, it } from 'vitest';

function collectStrings(value: unknown, out: string[] = []): string[] {
	if (typeof value === 'string') {
		out.push(value);
		return out;
	}

	if (Array.isArray(value)) {
		for (const item of value) {
			collectStrings(item, out);
		}
		return out;
	}

	if (value && typeof value === 'object') {
		for (const item of Object.values(value as Record<string, unknown>)) {
			collectStrings(item, out);
		}
		return out;
	}

	return out;
}

describe('caseStudies contract', () => {
	it('exports case studies with unique slugs', () => {
		expect(caseStudies.length).toBeGreaterThanOrEqual(1);
		const slugs = caseStudies.map((study) => study.meta.slug);
		expect(new Set(slugs).size).toBe(slugs.length);
	});

	it('includes required narrative + proof artifacts', () => {
		caseStudies.forEach((study) => {
			expect(study.meta.name.trim().length).toBeGreaterThan(0);
			expect(study.meta.tagline.trim().length).toBeGreaterThan(0);

			// Voice + narrative
			expect(study.hero.description).toMatch(/\bI\b/);
			expect(study.problem.body).toMatch(/\bI\b/);
			expect(study.solution.body).toMatch(/\bI\b/);

			// Proof points
			expect(study.diagrams?.architecture?.trim().length).toBeGreaterThan(0);
			expect(study.diagrams?.flow?.trim().length).toBeGreaterThan(0);
			expect(study.diagrams?.deployment?.trim().length).toBeGreaterThan(0);
			expect(study.skillsMatrix.length).toBeGreaterThanOrEqual(6);
			expect(study.deployment.target.trim().length).toBeGreaterThan(0);
			expect(study.deployment.runtime.trim().length).toBeGreaterThan(0);
			expect(study.deployment.ci.trim().length).toBeGreaterThan(0);
		});
	});

	it('does not mention disallowed palette colors', () => {
		const haystack = collectStrings(caseStudies).join('\n').toLowerCase();
		expect(haystack).not.toContain('teal');
		expect(haystack).not.toContain('purple');
	});
});
