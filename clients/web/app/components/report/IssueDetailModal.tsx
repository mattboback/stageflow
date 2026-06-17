import { useEffect, useMemo, useState } from 'react';

import type { IssueDetail, PageSummary } from '../../lib/types/unified-report';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import {
	generateContextualFix,
	getIssueKind,
	getIssueKindLabel,
	getIssueScreenshotUrl,
	getSeverityContainerClass,
	getSeverityDotClass,
	isManualReviewIssue,
	rewriteIssueTitle
} from '../../lib/report';

import { IssueEvidenceSection } from './IssueEvidenceSection';
import { IssueOccurrenceCard } from './IssueOccurrenceCard';

interface Props {
	issue: IssueDetail;
	page: PageSummary | null;
	issues: IssueDetail[];
	screenshots: ScreenshotArtifact[];
	onClose: () => void;
	onNavigate?: (issueId: string) => void;
}

type TabId = 'fix' | 'evidence' | 'occurrences';

export function IssueDetailModal({
	issue,
	page,
	issues,
	screenshots,
	onClose,
	onNavigate
}: Props) {
	const currentIndex = issues.findIndex((i) => i.id === issue.id);
	const totalCount = issues.length;
	const hasPrev = currentIndex > 0;
	const hasNext = currentIndex >= 0 && currentIndex < totalCount - 1;

	const issueKind = getIssueKind(issue);
	const kindLabel = getIssueKindLabel(issueKind);
	const isManual = isManualReviewIssue(issue);
	const displayTitle = rewriteIssueTitle(issue);
	const primaryOccurrence = issue.occurrences?.[0] ?? null;
	const occurrenceCount = issue.occurrences?.length ?? 0;
	const contextualFix = useMemo(
		() => generateContextualFix(issue, primaryOccurrence),
		[issue, primaryOccurrence]
	);
	const screenshotUrl = useMemo(
		() =>
			issue.pageId
				? getIssueScreenshotUrl({
						screenshots,
						scannerId: issue.scanner,
						issueId: issue.id,
						pageId: issue.pageId
					})
				: null,
		[screenshots, issue.scanner, issue.id, issue.pageId]
	);

	const hasEvidence = Boolean(
		screenshotUrl ||
			primaryOccurrence?.selector ||
			primaryOccurrence?.ancestorPath ||
			primaryOccurrence?.contextHtml ||
			primaryOccurrence?.html ||
			primaryOccurrence?.failureSummary
	);

	const availableTabs = useMemo<TabId[]>(() => {
		const tabs: TabId[] = ['fix'];
		if (hasEvidence) tabs.push('evidence');
		if (occurrenceCount > 0) tabs.push('occurrences');
		return tabs;
	}, [hasEvidence, occurrenceCount]);

	const [activeTab, setActiveTab] = useState<TabId>(availableTabs[0] ?? 'fix');

	useEffect(() => {
		if (!availableTabs.includes(activeTab)) {
			setActiveTab(availableTabs[0] ?? 'fix');
		}
	}, [availableTabs, activeTab]);

	useEffect(() => {
		setActiveTab(availableTabs[0] ?? 'fix');
		// reset when issue changes; intentionally only listening on issue.id
	}, [issue.id, availableTabs]);

	const navigate = (delta: number) => {
		if (!onNavigate || currentIndex < 0) return;
		const next = currentIndex + delta;
		if (next < 0 || next >= totalCount) return;
		const target = issues[next];
		if (target) onNavigate(target.id);
	};

	useEffect(() => {
		const handleKeydown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				event.preventDefault();
				onClose();
				return;
			}
			const target = event.target as HTMLElement | null;
			const tag = target?.tagName;
			if (tag === 'INPUT' || tag === 'TEXTAREA' || target?.isContentEditable) return;
			if (event.metaKey || event.ctrlKey || event.altKey) return;
			if (event.key === 'j' || event.key === 'ArrowDown') {
				if (hasNext) {
					event.preventDefault();
					navigate(1);
				}
			} else if (event.key === 'k' || event.key === 'ArrowUp') {
				if (hasPrev) {
					event.preventDefault();
					navigate(-1);
				}
			}
		};
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [hasNext, hasPrev, currentIndex, totalCount]);

	const pageLabel = page?.path ?? page?.url ?? null;

	return (
		<div
			className="imodal__backdrop"
			role="dialog"
			aria-modal="true"
			aria-label="Issue details"
			onClick={onClose}
		>
			<div className="imodal" onClick={(e) => e.stopPropagation()}>
				<header className={`imodal__head ${getSeverityContainerClass(issue.severity)}`}>
					<div className="imodal__head-top">
						<div className="imodal__chips">
							<span className={`imodal__sev sev-container sev-${issue.severity}`}>
								<span className={getSeverityDotClass(issue.severity)} aria-hidden="true" />
								{issue.severity}
							</span>
							<span className="imodal__kind">{kindLabel}</span>
							<span className="imodal__rule">
								{issue.scanner} · {issue.ruleId}
							</span>
						</div>
						<div className="imodal__nav">
							{onNavigate && totalCount > 1 && (
								<>
									<button
										type="button"
										className="imodal__nav-btn"
										onClick={() => navigate(-1)}
										disabled={!hasPrev}
										aria-label="Previous issue (k)"
										title="Previous (k)"
									>
										‹
									</button>
									<span className="imodal__nav-count">
										{currentIndex + 1} / {totalCount}
									</span>
									<button
										type="button"
										className="imodal__nav-btn"
										onClick={() => navigate(1)}
										disabled={!hasNext}
										aria-label="Next issue (j)"
										title="Next (j)"
									>
										›
									</button>
								</>
							)}
							<button
								type="button"
								className="imodal__close"
								onClick={onClose}
								aria-label="Close modal"
							>
								×
							</button>
						</div>
					</div>
					<h2 className="imodal__title">{displayTitle}</h2>
					{isManual && (
						<p className="imodal__sub">
							{issue.scanner === 'lighthouse' ? 'Lighthouse' : issue.scanner} flagged this
							for human verification — no concrete DOM target was reported.
						</p>
					)}
					{(pageLabel || issue.category || occurrenceCount > 0) && (
						<div className="imodal__strip">
							{pageLabel && (
								<span>
									<span className="imodal__strip-lab">Page</span>
									<span className="imodal__strip-val">{pageLabel}</span>
								</span>
							)}
							{issue.category && (
								<span>
									<span className="imodal__strip-lab">Category</span>
									<span className="imodal__strip-val">{issue.category}</span>
								</span>
							)}
							{occurrenceCount > 0 && (
								<span>
									<span className="imodal__strip-lab">Occurrences</span>
									<span className="imodal__strip-val">{occurrenceCount}</span>
								</span>
							)}
							{primaryOccurrence?.selector && (
								<span>
									<span className="imodal__strip-lab">Selector</span>
									<span className="imodal__strip-val imodal__strip-val--mono">
										{primaryOccurrence.selector}
									</span>
								</span>
							)}
						</div>
					)}
				</header>

				<nav className="imodal__tabs" role="tablist">
					{availableTabs.map((tab) => (
						<button
							key={tab}
							type="button"
							role="tab"
							aria-selected={activeTab === tab}
							className="imodal__tab"
							onClick={() => setActiveTab(tab)}
						>
							{tab === 'fix' && 'Fix'}
							{tab === 'evidence' && 'Evidence'}
							{tab === 'occurrences' && `Occurrences (${occurrenceCount})`}
						</button>
					))}
				</nav>

				<div className="imodal__body">
					{activeTab === 'fix' && (
						<div className="imodal__pane">
							<section>
								<h3 className="imodal__pane-h">How to fix</h3>
								{contextualFix ? (
									<p className="imodal__fix">{contextualFix}</p>
								) : (
									<p className="imodal__fix imodal__fix--muted">
										No fix guidance available for this rule yet.
									</p>
								)}
							</section>
							<section>
								<h3 className="imodal__pane-h">About this rule</h3>
								<p className="imodal__desc">{issue.description}</p>
								{issue.wcagTags && issue.wcagTags.length > 0 && (
									<div className="imodal__tags">
										{issue.wcagTags.map((tag) => (
											<span key={tag} className="imodal__tag">
												{tag}
											</span>
										))}
									</div>
								)}
							</section>
							{issue.helpUrl && (
								<a
									className="imodal__help"
									href={issue.helpUrl}
									target="_blank"
									rel="noopener noreferrer"
								>
									Learn more about this issue ↗
								</a>
							)}
						</div>
					)}
					{activeTab === 'evidence' && (
						<div className="imodal__pane">
							<IssueEvidenceSection
								issue={issue}
								page={page}
								screenshotUrl={screenshotUrl}
							/>
						</div>
					)}
					{activeTab === 'occurrences' && (
						<div className="imodal__pane">
							<h3 className="imodal__pane-h">Affected elements ({occurrenceCount})</h3>
							<div className="imodal__occs">
								{(issue.occurrences ?? []).map((occ, idx) => (
									<IssueOccurrenceCard
										key={idx}
										occurrence={occ}
										index={idx}
										page={page}
									/>
								))}
							</div>
						</div>
					)}
				</div>
			</div>
		</div>
	);
}
