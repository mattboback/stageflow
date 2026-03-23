import { alertVariants } from '$lib/components/ui/alert';
import { describe, expect, it } from 'vitest';

describe('alertVariants', () => {
	it('includes base structure classes', () => {
		expect(alertVariants({})).toContain('relative');
		expect(alertVariants({})).toContain('rounded-lg');
	});

	it('applies the variant classes', () => {
		expect(alertVariants({ variant: 'info' })).toContain('border-blue-200');
		expect(alertVariants({ variant: 'success' })).toContain('border-emerald-200');
		expect(alertVariants({ variant: 'warning' })).toContain('border-amber-200');
		expect(alertVariants({ variant: 'error' })).toContain('border-red-200');
	});
});
