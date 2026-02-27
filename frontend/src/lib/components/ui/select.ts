import { cva, type VariantProps } from 'class-variance-authority';

export const selectVariants = cva(
	'bg-surface-muted ring-offset-paper placeholder:text-ink-faint flex w-full rounded-lg border px-3 transition-colors disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2',
	{
		variants: {
			size: {
				sm: 'py-1.5 text-xs',
				md: 'py-2 text-sm'
			}
		},
		defaultVariants: {
			size: 'md'
		}
	}
);

export type SelectSize = VariantProps<typeof selectVariants>['size'];
