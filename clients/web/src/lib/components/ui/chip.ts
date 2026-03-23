import { type VariantProps, cva } from 'class-variance-authority';

export const chipVariants = cva(
	'inline-flex items-center justify-center rounded-full border font-semibold transition-colors',
	{
		variants: {
			tone: {
				default: 'border-line bg-surface text-ink-muted',
				muted: 'border-line bg-surface-muted text-ink-muted',
				active: 'border-ink bg-ink text-surface',
				success: 'border-emerald-200 bg-emerald-50 text-emerald-700',
				warning: 'border-amber-200 bg-amber-50 text-amber-700',
				danger: 'border-red-200 bg-red-50 text-red-700',
				ghost: 'border-transparent bg-transparent text-ink-muted'
			},
			size: {
				xs: 'px-2 py-0.5 text-xs',
				sm: 'px-3 py-1 text-xs',
				md: 'px-3 py-1 text-sm'
			},
			caps: {
				true: 'tracking-wide uppercase',
				false: ''
			},
			interactive: {
				true: 'cursor-pointer outline-none hover:bg-surface-muted hover:border-ink/20 hover:text-ink focus-visible:ring-accent focus-visible:ring-offset-paper focus-visible:ring-2 focus-visible:ring-offset-2',
				false: ''
			}
		},
		defaultVariants: {
			tone: 'default',
			size: 'sm',
			caps: false,
			interactive: false
		}
	}
);

export type ChipTone = VariantProps<typeof chipVariants>['tone'];
export type ChipSize = VariantProps<typeof chipVariants>['size'];
export type ChipCaps = VariantProps<typeof chipVariants>['caps'];
export type ChipInteractive = VariantProps<typeof chipVariants>['interactive'];
