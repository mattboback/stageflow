import { getPageOverviewUrl } from '../../lib/report/screenshots';
import { SEVERITY_LEVELS, type SeverityLevel } from '../../lib/report/severity';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import type { PageSummary, UnifiedReport } from '../../lib/types/unified-report';

interface Props {
	report: UnifiedReport;
	screenshots: ScreenshotArtifact[];
	onSelectPage: (pageId: string) => void;
}

/* Severities with at least one finding, worst first, as bar segments. */
function distribution(page: PageSummary): { severity: SeverityLevel; count: number }[] {
	const counts = page.bySeverity;
	if (!counts) return [];
	return SEVERITY_LEVELS.map((severity) => ({
		severity,
		count: counts[severity] ?? 0
	})).filter((segment) => segment.count > 0);
}

function label(page: PageSummary): string {
	if (page.path) return page.path;
	try {
		return new URL(page.url).pathname || '/';
	} catch {
		return page.url;
	}
}

/*
 * The site, rather than the findings.
 *
 * Everything here already existed in the report and had no surface: pages[]
 * carries per-page severity counts and a duration, and the page↔screenshot
 * join was already written for the visual review panel. What was missing was
 * the question "which page is the worst one", which is the first thing anybody
 * asks of a multi-page scan and which the flat findings list answers badly.
 *
 * ReportView hides this tab below two pages. A single-page scan has nothing to
 * compare, and a tab that always says "1" is furniture. It becomes a genuine
 * site map the day a crawler lands, which is a scanner-runner change, not this
 * one.
 */
export function PagesView({ report, screenshots, onSelectPage }: Props) {
	const worst = Math.max(...report.pages.map((page) => page.issueCount), 1);

	return (
		<ul className="pgrid">
			{report.pages.map((page) => {
				const shot = getPageOverviewUrl(screenshots, page.id);
				const segments = distribution(page);
				const markers = page.pageOverview?.elements.length ?? 0;

				return (
					<li key={page.id} className="pgrid__item">
						<button
							type="button"
							className="pcard"
							onClick={() => onSelectPage(page.id)}
							aria-label={`Review ${label(page)}, ${page.issueCount} findings`}
						>
							<span className="pcard__shot">
								{shot ? (
									/* Decorative: the accessible name is on the button, and a
									   second description of the same thumbnail is noise. */
									<img src={shot} alt="" loading="lazy" />
								) : (
									<span className="pcard__shot-empty">No screenshot</span>
								)}
							</span>

							<span className="pcard__body">
								<span className="pcard__path">{label(page)}</span>
								<span className="pcard__url">{page.url}</span>

								<span className="pcard__bar" aria-hidden="true">
									{segments.length === 0 ? (
										<span className="pcard__bar-clean" />
									) : (
										segments.map((segment) => (
											<span
												key={segment.severity}
												className={`pcard__bar-seg sev-${segment.severity}`}
												/* Width is share of the WORST page, not of this
												   page's own total, so the bars compare across
												   cards instead of all filling to 100%. */
												style={{ flexGrow: segment.count / worst }}
											/>
										))
									)}
								</span>

								<span className="pcard__meta">
									<span className="pcard__count num">{page.issueCount}</span>
									<span>{page.issueCount === 1 ? 'finding' : 'findings'}</span>
									{markers > 0 && (
										<span className="pcard__markers">· {markers} located on the page</span>
									)}
								</span>
							</span>
						</button>
					</li>
				);
			})}
		</ul>
	);
}
