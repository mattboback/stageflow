import { type VariantProps, cva } from "class-variance-authority";

export const panelVariants = cva("border-line bg-surface border", {
	variants: {
		variant: {
			default: "",
			muted: "bg-surface-muted",
			ghost: "",
		},
		padding: {
			none: "",
			xs: "p-2",
			sm: "p-4",
			md: "p-5",
			lg: "p-6",
			xl: "p-8",
		},
		rounded: {
			sm: "rounded-sm",
			md: "rounded-md",
			lg: "rounded-lg",
			xl: "rounded-xl",
			"2xl": "rounded-2xl",
			"3xl": "rounded-3xl",
		},
		interactive: {
			true: "hover-glow hover:border-ink/20 transition-all",
			false: "",
		},
	},
	defaultVariants: {
		variant: "default",
		padding: "lg",
		rounded: "md",
		interactive: false,
	},
});

export type PanelVariant = VariantProps<typeof panelVariants>["variant"];
export type PanelPadding = VariantProps<typeof panelVariants>["padding"];
export type PanelRounded = VariantProps<typeof panelVariants>["rounded"];
export type PanelInteractive = VariantProps<
	typeof panelVariants
>["interactive"];
