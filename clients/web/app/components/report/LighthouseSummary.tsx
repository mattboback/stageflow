import type { LighthouseCategorySummary } from '../../lib/types/unified-report';
import { scoreBandFor } from '../../lib/report';

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

export function LighthouseSummary({ categories }: Props) {
	if (!categories || categories.length === 0) {
		return <p className="lh__empty">No Lighthouse averages were collected.</p>;
	}

	return (
		<div className="lh__grid">
			{categories.map((cat) => {
				const raw = cat.avgScore;
				const score = typeof raw === 'number' ? Math.round(raw) : null;
				const band = scoreBandFor(score);
				const label = CATEGORY_LABELS[cat.id] ?? cat.title ?? cat.id;
				return (
					<div key={cat.id} className="lh__cell" data-tone={band?.tone ?? 'none'}>
						<p className="lh__cell-lab">{label}</p>
						<p className="lh__cell-score">{score !== null ? score : '—'}</p>
						{band ? <p className="lh__cell-band">{band.label}</p> : null}
					</div>
				);
			})}
		</div>
	);
}
