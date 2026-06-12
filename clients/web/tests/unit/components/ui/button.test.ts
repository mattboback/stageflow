import { buttonVariants } from '$lib/components/ui/button';
import { describe, expect, it } from 'vitest';

describe('buttonVariants', () => {
	it('includes base button classes', () => {
		expect(buttonVariants({})).toContain('inline-flex');
		expect(buttonVariants({})).toContain('focus-visible:ring-2');
	});

	it('applies size + variant classes', () => {
		expect(buttonVariants({ size: 'sm' })).toContain('h-8');
		expect(buttonVariants({ variant: 'outline' })).toContain('border-line');
		expect(buttonVariants({ variant: 'glow' })).toContain('bg-accent');
	});
});
