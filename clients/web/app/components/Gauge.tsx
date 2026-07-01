const R = 52;
const CIRC = 2 * Math.PI * R; // 326.7

/** Arc score dial. value 0-100. */
export function Gauge({
	value,
	caption = 'score',
	size = 132,
	valFontSize = '2.1rem',
	arcColor
}: {
	value: number;
	caption?: string;
	size?: number;
	valFontSize?: string;
	arcColor?: string;
}) {
	const offset = CIRC * (1 - Math.max(0, Math.min(100, value)) / 100);
	return (
		<div className="gauge">
			<svg
				className="dial"
				viewBox="0 0 120 120"
				width={size}
				height={size}
				role="img"
				aria-label={`${caption} ${value} of 100`}
			>
				<circle className="track" cx="60" cy="60" r={R} fill="none" strokeWidth="9" />
				<circle
					className="arc"
					cx="60"
					cy="60"
					r={R}
					fill="none"
					strokeWidth="9"
					strokeDasharray={CIRC.toFixed(1)}
					strokeDashoffset={offset.toFixed(1)}
					transform="rotate(-90 60 60)"
					style={arcColor ? { stroke: arcColor } : undefined}
				/>
			</svg>
			<div className="gauge__num">
				<span className="gauge__val" style={{ fontSize: valFontSize }}>
					{value}
				</span>
				<span className="gauge__cap">{caption}</span>
			</div>
		</div>
	);
}
