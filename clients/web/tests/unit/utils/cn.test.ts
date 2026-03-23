/**
 * cn (classnames) Utility Tests
 */

import { cn } from '$lib/utils/cn';
import { describe, expect, it } from 'vitest';

describe('cn', () => {
	describe('basic class merging', () => {
		it('merges multiple class strings', () => {
			const result = cn('class1', 'class2');
			expect(result).toBe('class1 class2');
		});

		it('handles single class', () => {
			const result = cn('single-class');
			expect(result).toBe('single-class');
		});

		it('handles empty inputs', () => {
			const result = cn();
			expect(result).toBe('');
		});

		it('handles empty string inputs', () => {
			const result = cn('', 'class1', '');
			expect(result).toBe('class1');
		});
	});

	describe('conditional classes', () => {
		it('handles boolean conditionals', () => {
			const isActive = Math.random() > -1; // always true, but TS doesn't know
			const isDisabled = Math.random() < 0; // always false, but TS doesn't know
			const result = cn('base', isActive ? 'active' : false, isDisabled ? 'disabled' : false);
			expect(result).toBe('base active');
		});

		it('handles undefined values', () => {
			const result = cn('base', undefined, 'extra');
			expect(result).toBe('base extra');
		});

		it('handles null values', () => {
			const result = cn('base', null, 'extra');
			expect(result).toBe('base extra');
		});

		it('handles false values', () => {
			const result = cn('base', false, 'extra');
			expect(result).toBe('base extra');
		});
	});

	describe('object syntax', () => {
		it('handles object with true values', () => {
			const result = cn({ active: true, disabled: false, hidden: true });
			expect(result).toBe('active hidden');
		});

		it('handles empty object', () => {
			const result = cn({});
			expect(result).toBe('');
		});

		it('combines object and string inputs', () => {
			const result = cn('base', { active: true, disabled: false });
			expect(result).toBe('base active');
		});
	});

	describe('array syntax', () => {
		it('handles array of classes', () => {
			const result = cn(['class1', 'class2']);
			expect(result).toBe('class1 class2');
		});

		it('handles nested arrays', () => {
			const result = cn(['class1', ['class2', 'class3']]);
			expect(result).toBe('class1 class2 class3');
		});

		it('handles array with conditionals', () => {
			const isHidden = Math.random() < 0; // always false, but TS doesn't know
			const result = cn(['base', isHidden ? 'hidden' : false, 'visible']);
			expect(result).toBe('base visible');
		});
	});

	describe('tailwind-merge behavior', () => {
		it('deduplicates conflicting tailwind classes', () => {
			const result = cn('p-4', 'p-2');
			expect(result).toBe('p-2');
		});

		it('merges conflicting margin classes', () => {
			const result = cn('m-2', 'm-4');
			expect(result).toBe('m-4');
		});

		it('preserves non-conflicting classes', () => {
			const result = cn('p-4', 'mt-2', 'mb-4');
			expect(result).toBe('p-4 mt-2 mb-4');
		});

		it('handles conflicting text colors', () => {
			const result = cn('text-red-500', 'text-blue-500');
			expect(result).toBe('text-blue-500');
		});

		it('handles conflicting background colors', () => {
			const result = cn('bg-white', 'bg-gray-100');
			expect(result).toBe('bg-gray-100');
		});

		it('handles responsive variants correctly', () => {
			const result = cn('md:p-4', 'md:p-6');
			expect(result).toBe('md:p-6');
		});

		it('preserves different responsive variants', () => {
			const result = cn('sm:p-2', 'md:p-4', 'lg:p-6');
			expect(result).toBe('sm:p-2 md:p-4 lg:p-6');
		});

		it('handles state variants correctly', () => {
			const result = cn('hover:bg-blue-500', 'hover:bg-red-500');
			expect(result).toBe('hover:bg-red-500');
		});

		it('preserves different state variants', () => {
			const result = cn('hover:bg-blue-500', 'focus:bg-red-500');
			expect(result).toBe('hover:bg-blue-500 focus:bg-red-500');
		});
	});

	describe('complex combinations', () => {
		it('handles complex real-world usage', () => {
			const isActive = true;
			const isDisabled = false;
			const result = cn('inline-flex items-center', 'px-4 py-2', 'rounded-md', {
				'bg-blue-500 text-white': isActive,
				'bg-gray-200 text-gray-500': isDisabled
			});
			expect(result).toContain('inline-flex');
			expect(result).toContain('items-center');
			expect(result).toContain('bg-blue-500');
			expect(result).toContain('text-white');
			expect(result).not.toContain('bg-gray-200');
		});

		it('handles variant classes from components', () => {
			const useSecondary = Math.random() < 0;
			const useSmall = Math.random() < 0;
			const variant: 'primary' | 'secondary' = useSecondary ? 'secondary' : 'primary';
			const size: 'small' | 'large' = useSmall ? 'small' : 'large';
			const result = cn(
				'btn',
				variant === 'primary' && 'btn-primary',
				variant === 'secondary' && 'btn-secondary',
				size === 'small' && 'btn-sm',
				size === 'large' && 'btn-lg'
			);
			expect(result).toBe('btn btn-primary btn-lg');
		});
	});
});
