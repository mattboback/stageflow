import { ArrowLeftRight } from 'lucide-react';

import {
	WCAG_THRESHOLDS,
	type ContrastLevel,
	contrastRatioFromStrings,
	formatRatio,
	parseColor,
	requiredLevel
} from '../../../lib/utils/contrast';

interface Props {
	fg: string;
	bg: string;
	ruleId: string;
	largeText: boolean;
	onFgChange: (value: string) => void;
	onBgChange: (value: string) => void;
	onSwap: () => void;
	onLargeTextChange: (value: boolean) => void;
}

const LEVELS: ContrastLevel[] = ['AA', 'AAA'];

export function ContrastResult({
	fg,
	bg,
	ruleId,
	largeText,
	onFgChange,
	onBgChange,
	onSwap,
	onLargeTextChange
}: Props) {
	const ratio = contrastRatioFromStrings(fg, bg);
	const required = requiredLevel(ruleId);
	const requiredThreshold = WCAG_THRESHOLDS[required][largeText ? 'large' : 'normal'];
	const requiredPasses = ratio !== null && ratio >= requiredThreshold;

	return (
		<div className="vfy__result">
			<div className="vfy__colors">
				<div className="vfy__color">
					<label className="vfy__color-lab" htmlFor="contrast-fg">
						Text color
					</label>
					<div className="vfy__color-row">
						<span
							className="vfy__swatch"
							style={parseColor(fg) ? { background: fg } : undefined}
							aria-hidden="true"
						/>
						<input
							id="contrast-fg"
							className="vfy__hex mono"
							value={fg}
							onChange={(e) => onFgChange(e.currentTarget.value)}
							placeholder="Not sampled"
							autoComplete="off"
							spellCheck={false}
						/>
					</div>
				</div>
				<button
					type="button"
					className="vfy__swap"
					onClick={onSwap}
					aria-label="Swap text and background colors"
					title="Swap colors"
				>
					<ArrowLeftRight size={15} aria-hidden="true" />
				</button>
				<div className="vfy__color">
					<label className="vfy__color-lab" htmlFor="contrast-bg">
						Background color
					</label>
					<div className="vfy__color-row">
						<span
							className="vfy__swatch"
							style={parseColor(bg) ? { background: bg } : undefined}
							aria-hidden="true"
						/>
						<input
							id="contrast-bg"
							className="vfy__hex mono"
							value={bg}
							onChange={(e) => onBgChange(e.currentTarget.value)}
							placeholder="Not sampled"
							autoComplete="off"
							spellCheck={false}
						/>
					</div>
				</div>
			</div>

			{/* Ratio is the headline result; the levels table qualifies it. */}
			<div className="vfy__hero">
				<span className="vfy__ratio" aria-live="polite">
					{ratio !== null ? (
						<>
							{formatRatio(ratio)}
							<small> : 1</small>
						</>
					) : (
						'—'
					)}
				</span>
				{ratio !== null && (
					<span
						className={`vfy__level-pill vfy__level-pill--${requiredPasses ? 'pass' : 'fail'}`}
					>
						{requiredPasses ? `Passes ${required}` : `Fails ${required}`}
					</span>
				)}
			</div>

			<table className="vfy__levels-table">
				<thead className="sr-only">
					<tr>
						<th scope="col">Level</th>
						<th scope="col">Required ratio</th>
						<th scope="col">Result</th>
					</tr>
				</thead>
				<tbody>
					{LEVELS.map((level) => {
						const threshold = WCAG_THRESHOLDS[level][largeText ? 'large' : 'normal'];
						const passes = ratio !== null && ratio >= threshold;
						const isRequired = level === required;
						return (
							<tr key={level} className={isRequired ? 'vfy__level-row--req' : undefined}>
								<td className="vfy__level-name">
									{level}
									{isRequired && <span className="vfy__level-req"> required</span>}
								</td>
								<td className="vfy__level-need num">≥ {threshold.toFixed(1)}:1</td>
								<td>
									{ratio !== null ? (
										<span
											className={`vfy__level-pill vfy__level-pill--${passes ? 'pass' : 'fail'}`}
										>
											{passes ? 'Pass' : 'Fail'}
										</span>
									) : (
										<span className="vfy__level-pill">No colors</span>
									)}
								</td>
							</tr>
						);
					})}
				</tbody>
			</table>

			<label className="vfy__large">
				<input
					type="checkbox"
					checked={largeText}
					onChange={(e) => onLargeTextChange(e.currentTarget.checked)}
				/>
				Large text (≥24px, or bold ≥18.7px)
			</label>
		</div>
	);
}
