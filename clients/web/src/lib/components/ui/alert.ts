import { type VariantProps, cva } from 'class-variance-authority';

export const alertVariants = cva(
	'relative w-full rounded-lg border px-4 py-3 text-sm [&>svg+div]:translate-y-px [&>svg]:h-4 [&>svg]:w-4 [&>svg]:text-current',
	{
		variants: {
			variant: {
				info: 'border-blue-200 bg-blue-50 text-blue-900',
				success: 'border-emerald-200 bg-emerald-50 text-emerald-900',
				warning: 'border-amber-200 bg-amber-50 text-amber-900',
				error: 'border-red-200 bg-red-50 text-red-900'
			}
		},
		defaultVariants: {
			variant: 'info'
		}
	}
);

export type AlertVariant = VariantProps<typeof alertVariants>['variant'];
