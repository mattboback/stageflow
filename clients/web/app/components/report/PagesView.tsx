import { useMemo, useState } from 'react';
import { getPageOverviewUrl } from '../../lib/report/screenshots';
import {
	describeSeverityCounts,
	severityCounts,
	severitySortIndex
} from '../../lib/report/severity';
import { getIssueKind } from '../../lib/report/issue-kind';
import { getSeverityBadgeClass } from '../../lib/report/severity';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import type { PageSummary, UnifiedReport } from '../../lib/types/unified-report';
import { SeverityBar, SeverityLegend } from './SeverityBar';

interface Props {
	report: UnifiedReport;
	screenshots: ScreenshotArtifact[];
	onSelectPage: (pageId: string) => void;
}

type PageSort = 'serious' | 'total' | 'alpha';

function label(page: PageSummary): string {
	if (page.path) return page.path;
	try {
		return new URL(page.url).pathname || '/';
	} catch {
		return page.url;
	}
}

function relativePath(page: PageSummary): string {
	try {
		const url = new URL(page.url);
		return `${url.pathname}${url.search}${url.hash}` || '/';
	} catch {
		return label(page);
	}
}

function formattedTitle(page: PageSummary): string {
	const path = relativePath(page).split(/[?#]/)[0] ?? '/';
	if (path === '/' || path === '') return 'Home';
	const slug =
		path
			.replace(/^\/+|\/+$/g, '')
			.split('/')
			.pop() ?? path;
	const words = slug.replace(/---+/g, ' ').replace(/-/g, ' ').trim();
	if (!words) return path;
	return words.charAt(0).toUpperCase() + words.slice(1);
}

export function PagesView({ report, screenshots, onSelectPage }: Props) {
	const [sort, setSort] = useState<PageSort>('serious');
	const worst = Math.max(...report.pages.map((page) => page.issueCount), 1);

	const sortedPages = useMemo(() => {
		const pages = [...report.pages];
		if (sort === 'alpha') return pages.sort((a, b) => label(a).localeCompare(label(b)));
		if (sort === 'total') return pages.sort((a, b) => b.issueCount - a.issueCount);
		return pages.sort((a, b) => {
			const aw = severityCounts(a.bySeverity)[0]?.severity;
			const bw = severityCounts(b.bySeverity)[0]?.severity;
			const bySeverity = severitySortIndex(aw) - severitySortIndex(bw);
			return bySeverity !== 0 ? bySeverity : b.issueCount - a.issueCount;
		});
	}, [report.pages, sort]);

	return (
		<>
			<div className="pgrid__toolbar">
				<label className="pgrid__sort">
					<span>Sort by</span>
					<select
						className="isidebar__select pgrid__sort-select"
						value={sort}
						onChange={(e) => setSort(e.target.value as PageSort)}
						aria-label="Sort pages"
					>
						<option value="serious">Most serious</option>
						<option value="total">Total findings</option>
						<option value="alpha">Alphabetical</option>
					</select>
				</label>
			</div>
			<ul className="pgrid">
				{sortedPages.map((page) => {
					const shot = getPageOverviewUrl(screenshots, page.id);
					const counts = severityCounts(page.bySeverity);
					const worstSev = counts[0]?.severity;
					const markers = page.pageOverview?.elements.length ?? 0;
					const pageIssues = report.issues.filter((issue) => issue.pageId === page.id);
					const onPageCount = pageIssues.filter(
						(issue) => getIssueKind(issue) === 'element'
					).length;
					const pageLevelCount = pageIssues.length - onPageCount;

					return (
						<li key={page.id} className="pgrid__item">
							<button
								type="button"
								className="pcard"
								onClick={() => onSelectPage(page.id)}
								aria-label={`Review ${label(page)}, ${page.issueCount} findings${counts.length > 0 ? `: ${describeSeverityCounts(counts)}` : ''}`}
							>
								<span className="pcard__shot">
									{shot ? (
										<img src={shot} alt="" loading="lazy" />
									) : (
										<span className="pcard__shot-empty">No screenshot</span>
									)}
								</span>

								<span className="pcard__body">
									<span className="pcard__head">
										<span className="pcard__path">{formattedTitle(page)}</span>
										{worstSev && (
											<span className={getSeverityBadgeClass(worstSev)}>{worstSev}</span>
										)}
									</span>
									<span className="pcard__url mono">{relativePath(page)}</span>

									<span className="pcard__meta">
										<span className="pcard__count num">{page.issueCount}</span>
										<span>{page.issueCount === 1 ? 'finding' : 'findings'}</span>
										{markers > 0 && (
											<span className="pcard__markers">· {markers} located on the page</span>
										)}
									</span>
									<span className="pcard__split">
										<span>{onPageCount} on-page</span>
										<span aria-hidden="true">·</span>
										<span>{pageLevelCount} page-level</span>
									</span>

									<SeverityBar counts={counts} scale={worst} className="pcard__bar" />
									{counts.length > 0 && <SeverityLegend counts={counts} inline />}
								</span>
							</button>
						</li>
					);
				})}
			</ul>
		</>
	);
}
