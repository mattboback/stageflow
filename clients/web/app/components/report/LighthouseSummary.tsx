import type { LighthouseCategorySummary } from '../../lib/types/unified-report';
import { scoreBandFor } from '../../lib/report';
import { Gauge } from '../Gauge';

interface Props {
	categories: LighthouseCategorySummary[];
}

const CATEGORY_LABELS: Record<string, string> = {
	performance: 'Performance',
	accessibility: 'Accessibility',
	'best-practices': 'Best Practices',
	seo: 'SEO',
	pwa: 'PWA'
};

/*
 * Lighthouse's category scores sat beside the site score and contradicted it
 * (93 / 99 / 98 next to an F). The caption says why: they grade Lighthouse's
 * own audits, while the site score counts every scanner.
 */
export function LighthouseSummary({ categories }: Props) {
	if (!categories || categories.length === 0) {
		return <p className="lh__empty">No Lighthouse averages were collected.</p>;
	}

	return (
		<section className="lh" aria-labelledby="lh-heading">
			<h2 id="lh-heading">Lighthouse averages</h2>
			<p className="lh__caption">
				Lighthouse’s own audits only; the site score also counts axe, security headers, links and
				SEO checks.
			</p>
			<ul className="lh__grid">
				{categories.map((cat) => {
					const raw = cat.avgScore;
					// The contract carries Lighthouse's native 0–1 score; bands are 0–100.
					const score = typeof raw === 'number' ? Math.round(raw * 100) : null;
					const band = scoreBandFor(score);
					const label = CATEGORY_LABELS[cat.id] ?? cat.title ?? cat.id;
					return (
						<li key={cat.id} className="lh__cell" data-tone={band?.tone ?? 'none'}>
							{score !== null ? (
								<Gauge
									value={score}
									caption=""
									size={76}
									valFontSize="1.25rem"
									ariaLabel={`${label} ${score} of 100${band ? `, ${band.label}` : ''}`}
								/>
							) : (
								<p className="lh__cell-score">—</p>
							)}
							<p className="lh__cell-lab">{label}</p>
							{band ? <p className="lh__cell-band">{band.label}</p> : null}
						</li>
					);
				})}
			</ul>
		</section>
	);
}
