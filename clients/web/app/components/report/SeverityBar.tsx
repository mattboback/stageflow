import type { SeverityCount, SeverityLevel } from '../../lib/report';
import { cn } from '../../lib/utils';

/*
 * Severity is a status scale, so neighbouring hues sit close together for
 * colour-blind readers by design. The bar never stands alone: every use puts
 * the counts in text beside it (SeverityLegend, or a summary line), and
 * segments are split by surface gaps instead of meeting edge to edge.
 */

interface SeverityBarProps {
	counts: readonly SeverityCount[];
	/**
	 * The total a full bar stands for. Defaults to this bar's own total; pass
	 * the largest of a set (the worst page) so bars compare across rows.
	 */
	scale?: number | undefined;
	/**
	 * Names the bar as an image. Without one it is hidden from assistive
	 * technology, and the text beside it carries the numbers.
	 */
	label?: string | undefined;
	size?: 'sm' | 'md' | 'lg';
	className?: string | undefined;
}

export function SeverityBar({ counts, scale, label, size = 'md', className }: SeverityBarProps) {
	const total = counts.reduce((sum, { count }) => sum + count, 0);
	const full = Math.max(scale ?? 0, total);
	return (
		<span
			className={cn('sevbar', `sevbar--${size}`, className)}
			{...(label ? { role: 'img', 'aria-label': label } : { 'aria-hidden': true })}
		>
			{total === 0 ? (
				<span className="sevbar__clean" title="No findings" />
			) : (
				<>
					{counts.map(({ severity, count }, index) => (
						<span
							key={severity}
							className={cn(
								'sevbar__seg',
								`sev-${severity}`,
								index === counts.length - 1 && 'sevbar__seg--end'
							)}
							style={{ flexGrow: count / full }}
							title={`${count.toLocaleString()} ${severity}`}
						/>
					))}
					{full > total && (
						<span className="sevbar__rest" style={{ flexGrow: (full - total) / full }} />
					)}
				</>
			)}
		</span>
	);
}

interface SeverityLegendProps {
	counts: readonly SeverityCount[];
	/** Makes each entry a button, such as a filter for that severity. */
	onSelect?: ((severity: SeverityLevel) => void) | undefined;
	/** A tooltip for each button, saying where it goes. */
	selectTitle?: ((severity: SeverityLevel) => string) | undefined;
	/** Plain text rather than a list, for use inside a button that already names it. */
	inline?: boolean;
	className?: string | undefined;
}

/** The counts behind a SeverityBar, as text: "3 critical · 14 serious". */
export function SeverityLegend({
	counts,
	onSelect,
	selectTitle,
	inline = false,
	className
}: SeverityLegendProps) {
	const entry = (severity: SeverityLevel, count: number) => (
		<>
			<span className={`sevlegend__swatch sev-${severity}`} aria-hidden="true" />
			<span className="num">{count.toLocaleString()}</span> {severity}
		</>
	);
	if (inline) {
		return (
			<span className={cn('sevlegend', className)}>
				{counts.map(({ severity, count }) => (
					<span key={severity} className="sevlegend__item">
						{entry(severity, count)}
					</span>
				))}
			</span>
		);
	}
	return (
		<ul className={cn('sevlegend', onSelect && 'sevlegend--buttons', className)}>
			{counts.map(({ severity, count }) => (
				<li key={severity}>
					{onSelect ? (
						<button
							type="button"
							className="sevlegend__item"
							onClick={() => onSelect(severity)}
							title={selectTitle?.(severity)}
						>
							{entry(severity, count)}
						</button>
					) : (
						<span className="sevlegend__item">{entry(severity, count)}</span>
					)}
				</li>
			))}
		</ul>
	);
}
