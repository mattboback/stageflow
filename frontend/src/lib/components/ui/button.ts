import { cva, type VariantProps } from 'class-variance-authority';

export const buttonVariants = cva(
	'inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-paper disabled:pointer-events-none disabled:opacity-50',
	{
		variants: {
			variant: {
				default: 'bg-accent text-white shadow-sm hover:bg-accent-hover',
				destructive: 'bg-red-600 text-white shadow-sm hover:bg-red-700',
				outline: 'border border-line bg-transparent text-ink shadow-sm hover:bg-surface-muted',
				secondary: 'bg-surface-muted text-ink shadow-sm hover:bg-surface',
				ghost: 'text-ink-muted hover:bg-surface-muted hover:text-ink',
				link: 'text-accent underline-offset-4 hover:underline',
				glow: 'bg-gradient-to-r from-accent to-accent-hover text-white shadow-md hover:brightness-105'
			},
			size: {
				default: 'h-10 px-4 py-2',
				sm: 'h-9 rounded-md px-3',
				lg: 'h-12 rounded-lg px-8 text-base',
				icon: 'h-10 w-10'
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
