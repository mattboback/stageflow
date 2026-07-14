/** Delta marker — the before/after motif: ▲ improvement, ▼ regression. */
export function Delta({
	value,
	improvedWhenPositive = true,
	label
}: {
	value: number;
	/** For scores, up is good; for issue counts, pass false so up reads as bad. */
	improvedWhenPositive?: boolean;
	label?: string;
}) {
	if (value === 0) {
		return (
			<span className="delta delta--flat num" aria-label={label ?? 'no change'}>
				— 0
			</span>
		);
	}
	const up = value > 0;
	const good = up === improvedWhenPositive;
	const arrow = up ? '▲' : '▼';
	const magnitude = Math.abs(value);
	return (
		<span
			className={`delta num ${good ? 'delta--good' : 'delta--bad'}`}
			aria-label={label ?? `${up ? 'up' : 'down'} ${magnitude}`}
		>
			{arrow} {magnitude}
		</span>
	);
}
