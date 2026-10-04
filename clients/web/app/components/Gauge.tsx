const R = 52;
const CIRC = 2 * Math.PI * R; // 326.7

const clampScore = (value: number) => Math.max(0, Math.min(100, value));

function describeChange(value: number, baseline: number): string {
	if (value === baseline) return `, unchanged from baseline ${baseline}`;
	const change = value - baseline;
	return `, ${change > 0 ? 'up' : 'down'} ${Math.abs(change)} from baseline ${baseline}`;
}

/** Arc score dial. value 0-100. */
export function Gauge({
	value,
	caption = 'score',
	size = 132,
	valFontSize = '2.1rem',
	arcColor,
	baseline,
	ariaLabel
}: {
	value: number;
	caption?: string;
	size?: number;
	valFontSize?: string;
	arcColor?: string;
	/**
	 * An earlier score to compare against: a tick on the ring where it was, and
	 * the points lost since then shaded between the two.
	 */
	baseline?: number | undefined;
	ariaLabel?: string;
}) {
	const offset = CIRC * (1 - clampScore(value) / 100);
	const lost =
		baseline !== undefined && baseline > value
			? (CIRC * (clampScore(baseline) - clampScore(value))) / 100
			: 0;
	return (
		<div className="gauge">
			<svg
				className="dial"
				viewBox="0 0 120 120"
				width={size}
				height={size}
				// Inline, so it wins over the .dial default.
				style={{ width: size, height: size }}
				role="img"
				aria-label={
					ariaLabel ??
					`${caption} ${value} of 100${baseline === undefined ? '' : describeChange(value, baseline)}`
				}
			>
				<circle className="track" cx="60" cy="60" r={R} fill="none" strokeWidth="9" />
				{lost > 0 && (
					<circle
						className="lost"
						cx="60"
						cy="60"
						r={R}
						fill="none"
						strokeWidth="9"
						strokeDasharray={`${lost.toFixed(1)} ${CIRC.toFixed(1)}`}
						strokeDashoffset={(-CIRC * (clampScore(value) / 100)).toFixed(1)}
						transform="rotate(-90 60 60)"
					/>
				)}
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
				{baseline !== undefined && (
					<g className="baseline" transform={`rotate(${clampScore(baseline) * 3.6} 60 60)`}>
						<line x1="60" y1={60 - R - 7} x2="60" y2={60 - R + 7} className="baseline__halo" />
						<line x1="60" y1={60 - R - 7} x2="60" y2={60 - R + 7} className="baseline__tick" />
					</g>
				)}
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
