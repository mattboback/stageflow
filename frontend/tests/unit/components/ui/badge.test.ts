import { badgeVariants } from '$lib/components/ui/badge';
import { describe, expect, it } from 'vitest';

describe('badgeVariants', () => {
	it('includes base classes', () => {
		expect(badgeVariants({})).toContain('inline-flex');
		expect(badgeVariants({})).toContain('rounded-md');
	});

	it('applies variants', () => {
		expect(badgeVariants({ variant: 'default' })).toContain('bg-accent');
		expect(badgeVariants({ variant: 'outline' })).toContain('border-line');
		expect(badgeVariants({ variant: 'terminal' })).toContain('font-mono');
	});
});
