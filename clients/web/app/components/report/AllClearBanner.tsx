import { CheckCircle2 } from 'lucide-react';

import type { UnifiedReport } from '../../lib/types/unified-report';
import { Pill } from '../Pill';

/* Positive zero-findings state for the review tab: without it, a clean scan
   ends in an empty review queue that reads as an anticlimax. */
export function AllClearBanner({ report }: { report: UnifiedReport }) {
	const pages = report.summary.pagesScanned;
	const scanners = report.scanners.length;
	const errorCount = report.errors?.length ?? 0;

	return (
		<div className="allclear panel">
			<div className="allclear__mark" aria-hidden="true">
				<CheckCircle2 size={30} />
			</div>
			<div className="allclear__copy">
				<div className="allclear__head">
					<h2>All clear.</h2>
					<Pill variant="done">0 findings</Pill>
				</div>
				<p>
					{scanners === 1 ? 'One scanner' : `All ${scanners} scanners`} completed across{' '}
					{pages === 1 ? 'one page' : `${pages.toLocaleString()} pages`} without reporting a single
					finding. A clean run makes a strong baseline for catching regressions later.
				</p>
				{errorCount > 0 && (
					<p className="allclear__caveat">
						{errorCount === 1 ? 'One report error was' : `${errorCount} report errors were`} noted
						during the run — check the Artifacts tab before promoting this run as a baseline.
					</p>
				)}
			</div>
			<a className="btn btn--ghost allclear__cta" href="#report-actions">
				Save this as a local baseline{' '}
				<span className="ar" aria-hidden="true">
					→
				</span>
			</a>
		</div>
	);
}
