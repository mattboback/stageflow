import { type VariantProps, cva } from "class-variance-authority";

export const badgeVariants = cva(
	"inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-accent focus:ring-offset-2 focus:ring-offset-paper",
	{
		variants: {
			variant: {
				default:
					"border-transparent bg-accent text-white hover:bg-accent-hover",
				secondary:
					"border-transparent bg-surface-muted text-ink hover:bg-surface",
				destructive:
					"border-transparent bg-red-600 text-white hover:bg-red-700",
				outline: "border-line text-ink",
				// Technical variants
				status:
					"rounded-sm border-ink/10 bg-surface font-mono text-[10px] tracking-wider uppercase",
				terminal:
					"rounded-sm border-accent/20 bg-accent/5 font-mono text-[10px] tracking-wider text-accent-ink uppercase",
				live: "rounded-sm border-emerald-300 bg-emerald-50 font-mono text-[10px] tracking-wider text-emerald-700 uppercase",
			},
		},
		defaultVariants: {
			variant: "default",
		},
	},
);

export type BadgeVariant = VariantProps<typeof badgeVariants>["variant"];
