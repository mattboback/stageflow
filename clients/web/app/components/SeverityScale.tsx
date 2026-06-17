export interface SeverityCounts {
	critical: number;
	serious: number;
	moderate: number;
	minor: number;
}

const ROWS: { key: keyof SeverityCounts; label: string }[] = [
	{ key: 'critical', label: 'Critical' },
	{ key: 'serious', label: 'Serious' },
	{ key: 'moderate', label: 'Moderate' },
	{ key: 'minor', label: 'Minor' }
];

/** Calibrated severity scale. Color is always paired with a label (color-blind safe). */
export function SeverityScale({ counts, style }: { counts: SeverityCounts; style?: React.CSSProperties }) {
	return (
		<div className="sevscale" style={style}>
			{ROWS.map(({ key, label }) => (
				<div className={`sevrow sev-${key}`} key={key}>
					<span className="sevrow__dot" />
					<span className="sevrow__name">{label}</span>
					<span className="sevrow__count">{counts[key]}</span>
				</div>
			))}
		</div>
	);
}
