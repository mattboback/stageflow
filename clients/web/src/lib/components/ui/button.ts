import { type VariantProps, cva } from 'class-variance-authority';

export const buttonVariants = cva(
	'inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-paper disabled:pointer-events-none disabled:opacity-50 transition-colors',
	{
		variants: {
			variant: {
				default: 'bg-accent text-white hover:bg-accent-hover',
				destructive: 'bg-red-600 text-white hover:bg-red-700',
				outline: 'border border-line bg-transparent text-ink hover:bg-surface-muted',
				secondary: 'bg-surface-muted text-ink hover:bg-line',
				ghost: 'text-ink-muted hover:bg-surface-muted hover:text-ink',
				link: 'text-accent underline-offset-4 hover:underline',
				glow: 'bg-accent text-white hover:bg-accent-hover'
			},
			size: {
				default: 'h-9 px-4 py-2',
				sm: 'h-8 px-3 text-xs',
				lg: 'h-11 px-6',
				icon: 'h-9 w-9'
			}
		},
		defaultVariants: {
			variant: 'default',
			size: 'default'
		}
	}
);

export type ButtonVariant = VariantProps<typeof buttonVariants>['variant'];
export type ButtonSize = VariantProps<typeof buttonVariants>['size'];
