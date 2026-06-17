import type { SeverityCounts } from '../../lib/types/unified-report';
import { SEVERITY_LEVELS, getSeverityDotClass } from '../../lib/report';

interface Props {
	bySeverity: SeverityCounts | undefined;
}

const SEV_LABEL: Record<string, string> = {
	critical: 'Critical',
	serious: 'Serious',
	moderate: 'Moderate',
	minor: 'Minor',
	info: 'Info'
};

export function SeverityBreakdown({ bySeverity }: Props) {
	const counts: Record<string, number> = bySeverity
		? { ...bySeverity }
		: {};
	const total = SEVERITY_LEVELS.reduce((sum, k) => sum + (counts[k] ?? 0), 0);

	return (
		<ul className="sbreak" aria-label="Issues by severity">
			{SEVERITY_LEVELS.map((sev) => {
				const value = counts[sev] ?? 0;
				const fraction = total > 0 ? value / total : 0;
				return (
					<li key={sev} className="sbreak__row">
						<span className="sbreak__label">
							<span className={getSeverityDotClass(sev)} aria-hidden="true" />
							{SEV_LABEL[sev]}
						</span>
						<span className="sbreak__bar" aria-hidden="true">
							<span
								className="sbreak__bar-fill"
								data-sev={sev}
								style={{ width: '100%', transform: `scaleX(${fraction})` }}
							/>
						</span>
						<span className="sbreak__count">{value.toLocaleString()}</span>
					</li>
				);
			})}
		</ul>
	);
}
