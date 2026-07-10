import type { SeverityCounts } from '../../lib/types/unified-report';
import { SEVERITY_LEVELS } from '../../lib/report';

interface Props {
	counts: SeverityCounts | undefined;
	centerLabel?: string;
	size?: number;
}

const SEV_COLORS: Record<string, string> = {
	critical: 'var(--sev-critical, #dc2626)',
	serious: 'var(--sev-serious, #ea580c)',
	moderate: 'var(--sev-moderate, #d97706)',
	minor: 'var(--sev-minor, #2563eb)',
	info: 'var(--sev-pass, #16a34a)'
};

export function SeverityDonut({ counts, centerLabel = 'issues', size = 132 }: Props) {
	const safe: Record<string, number> = counts ? { ...counts } : {};
	const total = SEVERITY_LEVELS.reduce((sum, k) => sum + (safe[k] ?? 0), 0);
	const radius = size / 2 - 14;
	const circumference = 2 * Math.PI * radius;

	if (total === 0) {
		return (
			<svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} aria-label="No issues">
				<circle
					cx={size / 2}
					cy={size / 2}
					r={radius}
					fill="none"
					stroke="var(--line)"
					strokeWidth={14}
				/>
				<text
					x="50%"
					y="50%"
					textAnchor="middle"
					dominantBaseline="central"
					fill="var(--ink-strong)"
					fontFamily="var(--display)"
					fontWeight="700"
					fontSize="1.1rem"
				>
					0
				</text>
			</svg>
		);
	}

	const segments = SEVERITY_LEVELS.flatMap((sev, index) => {
		const value = safe[sev] ?? 0;
		if (value === 0) return [];
		const fraction = value / total;
		const dash = fraction * circumference;
		const offset = SEVERITY_LEVELS.slice(0, index).reduce(
			(sum, previous) => sum + ((safe[previous] ?? 0) / total) * circumference,
			0
		);
		const segment = (
			<circle
				key={sev}
				cx={size / 2}
				cy={size / 2}
				r={radius}
				fill="none"
				stroke={SEV_COLORS[sev] ?? 'var(--ink-muted)'}
				strokeWidth={14}
				strokeDasharray={`${dash} ${circumference - dash}`}
				strokeDashoffset={-offset}
				transform={`rotate(-90 ${size / 2} ${size / 2})`}
			/>
		);
		return [segment];
	});

	return (
		<svg
			width={size}
			height={size}
			viewBox={`0 0 ${size} ${size}`}
			aria-label={`${total} ${centerLabel}`}
		>
			<circle
				cx={size / 2}
				cy={size / 2}
				r={radius}
				fill="none"
				stroke="var(--line)"
				strokeWidth={14}
			/>
			{segments}
			<text
				x="50%"
				y="46%"
				textAnchor="middle"
				dominantBaseline="central"
				fill="var(--ink-strong)"
				fontFamily="var(--display)"
				fontWeight="700"
				fontSize="1.5rem"
			>
				{total.toLocaleString()}
			</text>
			<text
				x="50%"
				y="63%"
				textAnchor="middle"
				dominantBaseline="central"
				fill="var(--ink-muted)"
				fontFamily="var(--sans)"
				fontSize="0.72rem"
				fontWeight="600"
			>
				{centerLabel}
			</text>
		</svg>
	);
}
