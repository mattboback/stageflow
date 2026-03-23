import { selectVariants } from '$lib/components/ui/select';
import { describe, expect, it } from 'vitest';

describe('selectVariants', () => {
	it('defaults to md sizing', () => {
		const cls = selectVariants();
		expect(cls).toContain('text-sm');
	});

	it('supports sm sizing', () => {
		const cls = selectVariants({ size: 'sm' });
		expect(cls).toContain('text-xs');
	});
});
