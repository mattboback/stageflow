export type StatusPillTone = 'strong' | 'watch' | 'needs-work' | 'high-risk' | 'failing';

export interface ScoreBand {
	tone: StatusPillTone;
	label: string;
}

export function scoreBandFor(score: number | null | undefined): ScoreBand | null {
	if (score === null || score === undefined || Number.isNaN(score)) return null;
	if (score >= 90) return { tone: 'strong', label: 'Strong' };
	if (score >= 80) return { tone: 'watch', label: 'Watch' };
	if (score >= 70) return { tone: 'needs-work', label: 'Needs work' };
	if (score >= 60) return { tone: 'high-risk', label: 'High risk' };
	return { tone: 'failing', label: 'Failing' };
}
