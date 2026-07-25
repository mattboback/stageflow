type PillVariant = 'live' | 'done' | 'queued' | 'error';

/** Status pill with LED. */
export function Pill({
	variant,
	children,
	led = true,
	ledColor,
	style
}: {
	variant?: PillVariant;
	children: React.ReactNode;
	led?: boolean;
	ledColor?: string;
	style?: React.CSSProperties;
}) {
	return (
		<span className={variant ? `pill pill--${variant}` : 'pill'} style={style}>
			{led && (
				<span className="pill__led" style={ledColor ? { background: ledColor } : undefined} />
			)}
			{children}
		</span>
	);
}
