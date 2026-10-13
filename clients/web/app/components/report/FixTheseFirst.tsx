import { ChevronRight } from 'lucide-react';

import {
	compareSeverity,
	findOverviewElement,
	generateContextualFix,
	getPageOverviewUrl,
	getSeverityBadgeClass,
	groupIssuesByRule,
	humanizeFailureSummary,
	needsHumanReview,
	scannerLabel,
	scoreIfFixed,
	sortIssues,
	type ScoreModel
} from '../../lib/report';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import type { IssueDetail, PageSummary } from '../../lib/types/unified-report';
import { ElementCrop } from './ElementCrop';
import { ScannerText } from './ScannerText';

interface Props {
	issues: IssueDetail[];
	onIssueSelect: (issue: IssueDetail) => void;
	newIssueIds?: ReadonlySet<string> | undefined;
	/** From explainScore. With it, each fix shows what it is worth. */
	scoreModel?: ScoreModel | null | undefined;
	/** With screenshots, a located finding shows its element. */
	pages?: readonly PageSummary[] | undefined;
	screenshots?: ScreenshotArtifact[] | undefined;
}

const ALL_OF = ['', 'this', 'both', 'all three', 'all four', 'all five'];

export function FixTheseFirst({
	issues,
	onIssueSelect,
	newIssueIds,
	scoreModel,
	pages,
	screenshots
}: Props) {
	const groups = groupIssuesByRule(issues.filter((issue) => !needsHumanReview(issue)))
		.sort(
			(a, b) =>
				compareSeverity(a.severity, b.severity) ||
				b.pageIds.length - a.pageIds.length ||
				a.fingerprint.localeCompare(b.fingerprint)
		)
		.slice(0, 5);
	if (groups.length === 0) return null;

	const idsOf = (group: (typeof groups)[number]) => group.occurrences.map((issue) => issue.id);
	// Computed together, never summed: the curve flattens, so gains do not add.
	const together = scoreModel ? scoreIfFixed(scoreModel, groups.flatMap(idsOf)) : null;
	const rows = groups.flatMap((group) => {
		const issue = sortIssues(group.occurrences, 'severity')[0];
		if (!issue) return [];
		const page = pages?.find((candidate) => candidate.id === issue.pageId) ?? null;
		const overviewUrl = page && screenshots ? getPageOverviewUrl(screenshots, page.id) : null;
		const element = findOverviewElement(page, issue.id);
		return [
			{
				group,
				issue,
				fix: humanizeFailureSummary(generateContextualFix(issue, issue.occurrences?.[0] ?? null)),
				gain: scoreModel ? scoreIfFixed(scoreModel, idsOf(group)) - scoreModel.score : 0,
				crop: page && overviewUrl && element ? { page, overviewUrl, element } : null
			}
		];
	});
	const withThumbs = rows.some((row) => row.crop);

	return (
		<section className="fix-first" aria-labelledby="fix-first-heading">
			<h2 id="fix-first-heading">Fix these first</h2>
			<p>Confirmed findings, highest severity and widest page coverage first.</p>
			{scoreModel && together !== null && together > scoreModel.score && (
				<p className="fix-first__projection">
					Fixing {ALL_OF[groups.length]} would take the score from{' '}
					<strong className="num">{scoreModel.score}</strong> to{' '}
					<strong className="num">{together}</strong>.
				</p>
			)}
			<ol className={`fix-first__list${withThumbs ? ' fix-first__list--thumbs' : ''}`}>
				{rows.map(({ group, issue, fix, gain, crop }, index) => (
					<li key={group.fingerprint}>
						<button type="button" className="fix-first__item" onClick={() => onIssueSelect(issue)}>
							<span className="fix-first__rank num" aria-hidden="true">
								{index + 1}
							</span>
							{withThumbs && (
								// The thumbnail repeats what the title says; the evidence tab
								// is where the element is described.
								<span className="fix-first__thumb" aria-hidden="true">
									{crop ? (
										<ElementCrop
											className="fix-first__crop"
											page={crop.page}
											overviewUrl={crop.overviewUrl}
											element={crop.element}
											ariaLabel=""
										/>
									) : (
										<span className="fix-first__crop fix-first__crop--none">Page-wide</span>
									)}
								</span>
							)}
							<span className="fix-first__body">
								<span className="fix-first__title">
									{group.title}{' '}
									{group.occurrences.some((item) => newIssueIds?.has(item.id)) && (
										<span className="issue-new">NEW</span>
									)}
								</span>
								<span className="fix-first__meta">
									<span className={getSeverityBadgeClass(group.severity)}>{group.severity}</span>
									<span>
										{group.pageIds.length} affected page{group.pageIds.length === 1 ? '' : 's'} ·{' '}
										{group.occurrences.length} occurrence{group.occurrences.length === 1 ? '' : 's'}{' '}
										· {scannerLabel(group.scanner)}
									</span>
								</span>
								<span className="fix-first__guidance">
									<ScannerText
										text={fix || 'Open this finding to inspect the evidence and rule guidance.'}
										links={false}
									/>
								</span>
							</span>
							{gain > 0 && (
								<span className="fix-first__gain num">
									+{gain}
									<span className="sr-only"> points on its own</span>
								</span>
							)}
							<ChevronRight className="fix-first__open" size={18} aria-hidden="true" />
						</button>
					</li>
				))}
			</ol>
		</section>
	);
}
