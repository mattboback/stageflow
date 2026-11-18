import './ScoreHistory.css';

interface ScorePoint {
	/** The run's job id. */
	id: string;
	score: number;
	/** When the run finished, as an ISO timestamp. */
	at: string;
}

const WIDTH = 132;
const HEIGHT = 40;
// A marker's radius plus its ring, so the end markers are not clipped.
const PAD = 6;
const MIN_SPAN = 10;

/**
 * The scores the chart's height covers. Never fewer than MIN_SPAN points, so a
 * one-point change reads as a small step rather than the full height.
 */
function scoreDomain(scores: number[]): [number, number] {
	const low = Math.min(...scores);
	const high = Math.max(...scores);
	if (high - low >= MIN_SPAN) return [low, high];
	const floor = Math.max(0, Math.min(100 - MIN_SPAN, (low + high - MIN_SPAN) / 2));
	return [floor, floor + MIN_SPAN];
}

function formatDay(iso: string): string {
	const date = new Date(iso);
	return Number.isNaN(date.getTime())
		? 'Unknown date'
		: date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

/**
 * A line of recent run scores, oldest first. The baseline run is a hollow
 * marker on a hairline at the baseline score, and the latest run a filled one.
 * Hovering a run shows its date and score.
 */
export function ScoreHistory({
	points,
	baselineId,
	baselineScore
}: {
	points: ScorePoint[];
	baselineId?: string | undefined;
	baselineScore?: number | undefined;
}) {
	if (points.length < 2) return null;
	const scores = points.map((point) => point.score);
	const [low, high] = scoreDomain(
		baselineScore === undefined ? scores : [...scores, baselineScore]
	);
	const step = (WIDTH - 2 * PAD) / (points.length - 1);
	const x = (index: number) => PAD + index * step;
	const y = (score: number) =>
		Number((PAD + ((high - score) / (high - low)) * (HEIGHT - 2 * PAD)).toFixed(1));
	const last = points.length - 1;
	const described = points
		.map((point) => `${point.score}${point.id === baselineId ? ' (baseline)' : ''}`)
		.join(', ');

	return (
		<svg
			className="score-history"
			viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
			width={WIDTH}
			height={HEIGHT}
			role="img"
			aria-label={`Scores of the last ${points.length} runs, oldest first: ${described}`}
		>
			{baselineScore !== undefined && (
				<line
					className="score-history__baseline"
					x1={0}
					x2={WIDTH}
					y1={y(baselineScore)}
					y2={y(baselineScore)}
				/>
			)}
			<polyline
				className="score-history__line"
				points={points.map((point, index) => `${x(index)},${y(point.score)}`).join(' ')}
			/>
			{points.map((point, index) => {
				const isBaseline = point.id === baselineId;
				const kind = isBaseline ? 'baseline' : index === last ? 'latest' : 'run';
				// Each run owns the strip of width nearest to it, so the pointer
				// only has to be closest to a run, not on its marker.
				const left = index === 0 ? 0 : x(index) - step / 2;
				const right = index === last ? WIDTH : x(index) + step / 2;
				return (
					<g key={point.id} className={`score-history__run score-history__run--${kind}`}>
						<title>
							{`${formatDay(point.at)} · score ${point.score}${isBaseline ? ' · baseline' : ''}${index === last ? ' · latest' : ''}`}
						</title>
						<rect
							className="score-history__hit"
							x={left}
							y={0}
							width={right - left}
							height={HEIGHT}
						/>
						<circle className="score-history__dot" cx={x(index)} cy={y(point.score)} r={4} />
					</g>
				);
			})}
		</svg>
	);
}
