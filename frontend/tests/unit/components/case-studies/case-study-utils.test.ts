import {
	endpointBadgeClass,
	endpointBadgeText,
	splitParagraphs,
	stateClass
} from '$lib/components/case-studies/case-study-utils';
import { describe, expect, it } from 'vitest';

describe('case-study-utils', () => {
	it('maps state classes', () => {
		expect(stateClass('FAILED')).toBe('bg-rose-500/20 text-rose-700');
		expect(stateClass('SUCCESS')).toBe('bg-emerald-500/20 text-emerald-700');
		expect(stateClass('DONE')).toBe('bg-accent-soft text-accent-ink');
		expect(stateClass('COMPLETE')).toBe('bg-accent-soft text-accent-ink');
		expect(stateClass('REPORT_GENERATION')).toBe('bg-accent-soft text-accent-ink');
		expect(stateClass('RUNNING')).toBe('border border-line bg-surface text-ink-muted');
	});

	it('formats endpoint badge text', () => {
		expect(endpointBadgeText('cli')).toBe('$');
		expect(endpointBadgeText(' python ')).toBe('py');
		expect(endpointBadgeText('get')).toBe('GET');
		expect(endpointBadgeText('POST')).toBe('POST');
	});

	it('maps endpoint badge classes', () => {
		expect(endpointBadgeClass('cli')).toBe('bg-ink text-white');
		expect(endpointBadgeClass('post')).toBe('bg-emerald-100 text-emerald-700');
		expect(endpointBadgeClass('get')).toBe('bg-blue-100 text-blue-700');
		expect(endpointBadgeClass('python')).toBe('bg-blue-100 text-blue-700');
		expect(endpointBadgeClass('PATCH')).toBe('border border-line bg-surface-muted text-ink-muted');
	});

	it('splits paragraphs on blank lines', () => {
		expect(splitParagraphs('a\n\nb')).toEqual(['a', 'b']);
		expect(splitParagraphs(' a \n\n\n b \n\n')).toEqual(['a', 'b']);
		expect(splitParagraphs('single paragraph')).toEqual(['single paragraph']);
	});
});
