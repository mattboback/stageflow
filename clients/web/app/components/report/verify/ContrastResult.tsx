import { ArrowLeftRight } from 'lucide-react';

import type { ContrastVerdict } from '../../../lib/hooks/useContrastVerdicts';
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
	verdict: ContrastVerdict | null;
	onFgChange: (value: string) => void;
	onBgChange: (value: string) => void;
	onSwap: () => void;
	onLargeTextChange: (value: boolean) => void;
	onRecord: (verdict: 'pass' | 'fail', ratio: number | null) => void;
	onClear: () => void;
}

const LEVELS: ContrastLevel[] = ['AA', 'AAA'];

function formatVerdictTime(iso: string): string {
	const date = new Date(iso);
	if (Number.isNaN(date.getTime())) return iso;
	return date.toLocaleString(undefined, {
		month: 'short',
		day: 'numeric',
		hour: 'numeric',
		minute: '2-digit'
	});
}

export function ContrastResult({
	fg,
	bg,
	ruleId,
	largeText,
	verdict,
	onFgChange,
	onBgChange,
	onSwap,
	onLargeTextChange,
	onRecord,
	onClear
}: Props) {
	const ratio = contrastRatioFromStrings(fg, bg);
	const required = requiredLevel(ruleId);

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
							placeholder="#1f2933"
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
					<ArrowLeftRight size={14} aria-hidden="true" />
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
							placeholder="#ffffff"
							autoComplete="off"
							spellCheck={false}
						/>
					</div>
				</div>
			</div>

			<div className="vfy__readout">
				<div className="vfy__ratio-box">
					<span className="vfy__ratio-lab">Measured ratio</span>
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
				</div>
				<ul className="vfy__levels">
					{LEVELS.map((level) => {
						const threshold = WCAG_THRESHOLDS[level][largeText ? 'large' : 'normal'];
						const passes = ratio !== null && ratio >= threshold;
						return (
							<li key={level} className="vfy__level">
								<span
									className={`vfy__level-lab${level === required ? ' vfy__level-lab--req' : ''}`}
								>
									{level} · {threshold.toFixed(1)}:1
								</span>
								{ratio !== null ? (
									<span
										className={`vfy__level-pill vfy__level-pill--${passes ? 'pass' : 'fail'}`}
									>
										{passes ? 'Pass' : 'Fail'}
									</span>
								) : (
									<span className="vfy__level-pill">No colors</span>
								)}
								{level === required && (
									<span className="vfy__level-req">required</span>
								)}
							</li>
						);
					})}
				</ul>
				<label className="vfy__large">
					<input
						type="checkbox"
						checked={largeText}
						onChange={(e) => onLargeTextChange(e.currentTarget.checked)}
					/>
					Large text (≥24px, or bold ≥18.7px)
				</label>
			</div>

			<div className="vfy__verdict">
				{verdict ? (
					<div className="vfy__verdict-row">
						<span
							className={`vfy__level-pill vfy__level-pill--${verdict.verdict === 'pass' ? 'pass' : 'fail'}`}
						>
							Verified · {verdict.verdict}
						</span>
						<span className="vfy__verdict-meta mono">
							{verdict.fg || '—'} on {verdict.bg || '—'}
							{verdict.ratio !== null ? ` · ${formatRatio(verdict.ratio)}:1` : ''} ·{' '}
							{formatVerdictTime(verdict.at)}
						</span>
						<button type="button" className="vfy__btn vfy__btn--ghost" onClick={onClear}>
							Clear verdict
						</button>
					</div>
				) : (
					<div className="vfy__verdict-row">
						<span className="vfy__verdict-ask">Does this element pass in context?</span>
						<button
							type="button"
							className="vfy__btn vfy__btn--pass"
							onClick={() => onRecord('pass', ratio)}
						>
							Mark pass
						</button>
						<button
							type="button"
							className="vfy__btn vfy__btn--fail"
							onClick={() => onRecord('fail', ratio)}
						>
							Mark fail
						</button>
					</div>
				)}
			</div>
		</div>
	);
}
