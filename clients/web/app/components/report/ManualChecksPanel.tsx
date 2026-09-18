import { ScannerText } from './ScannerText';
import type { ManualCheck } from '../../lib/report';

interface Props {
	checks: ManualCheck[];
	pagesScanned: number;
}

export function ManualChecksPanel({ checks, pagesScanned }: Props) {
	if (checks.length === 0) return null;

	return (
		<details className="mchecks">
			<summary className="mchecks__head">
				<h3>Manual checks</h3>
				<span className="mchecks__count">{checks.length}</span>
			</summary>
			<p className="mchecks__lede">
				Lighthouse cannot automate these, so it lists them for every page it audits. They are a
				checklist for a person to work through, not findings about this site, and they are not
				counted above.
			</p>
			<ul className="mchecks__list">
				{checks.map((check) => (
					<li key={check.ruleId} className="mchecks__row">
						<p className="mchecks__title">Verify: {check.title}</p>
						<p className="mchecks__desc">
							<ScannerText text={check.description} />
						</p>
						<p className="mchecks__meta">
							lighthouse · {check.ruleId}
							{check.pageCount < pagesScanned && (
								<>
									{' '}
									· {check.pageCount} of {pagesScanned} pages
								</>
							)}
						</p>
					</li>
				))}
			</ul>
		</details>
	);
}
