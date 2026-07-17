import { useEffect, useMemo, useState, useRef } from 'react';

import type { IssueDetail, PageSummary } from '../../lib/types/unified-report';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import {
	generateContextualFix,
	getIssueKind,
	getIssueKindLabel,
	getIssueScreenshotUrl,
	getPageOverviewUrl,
	getSeverityDotClass,
	normalizeSeverity,
	isAxeIncompleteIssue,
	isColorContrastIssue,
	isManualReviewIssue,
	needsHumanReview,
	rewriteIssueTitle
} from '../../lib/report';
import { useReviewVerdicts } from '../../lib/hooks/useReviewVerdicts';

import { IssueEvidenceSection } from './IssueEvidenceSection';
import { IssueOccurrenceCard } from './IssueOccurrenceCard';
import { ScannerText } from './ScannerText';
import { ManualReviewTab } from './verify/ManualReviewTab';
import { VerifyContrastTab } from './verify/VerifyContrastTab';

interface Props {
	issue: IssueDetail;
	jobId?: string;
	page: PageSummary | null;
	issues: IssueDetail[];
	screenshots: ScreenshotArtifact[];
	onClose: () => void;
	onNavigate?: (issueId: string) => void;
}

type TabId = 'review' | 'fix' | 'evidence' | 'verify' | 'occurrences';

export function IssueDetailModal({
	issue,
	jobId = '',
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

	const pageOverviewUrl = useMemo(
		() =>
			page
				? getPageOverviewUrl(screenshots, page.id, issue.scanner ? [issue.scanner, 'axe'] : ['axe'])
				: null,
		[screenshots, page, issue.scanner]
	);

	const hasOverviewCrop = Boolean(
		pageOverviewUrl && (page?.pageOverview?.elements ?? []).some((el) => el.issueId === issue.id)
	);

	const hasEvidence = Boolean(
		screenshotUrl ||
		hasOverviewCrop ||
		primaryOccurrence?.selector ||
		primaryOccurrence?.ancestorPath ||
		primaryOccurrence?.contextHtml ||
		primaryOccurrence?.html ||
		primaryOccurrence?.failureSummary
	);

	const canVerifyContrast = isColorContrastIssue(issue);
	const needsVerification = canVerifyContrast && isAxeIncompleteIssue(issue);
	const needsReview = needsHumanReview(issue);
	const { getVerdict } = useReviewVerdicts(jobId);
	const reviewVerdict = canVerifyContrast || needsReview ? getVerdict(issue.id) : null;

	const availableTabs = useMemo<TabId[]>(() => {
		const tabs: TabId[] = isManual ? ['review', 'fix'] : ['fix'];
		if (hasEvidence) tabs.push('evidence');
		if (canVerifyContrast) tabs.push('verify');
		if (occurrenceCount > 0) tabs.push('occurrences');
		return tabs;
	}, [isManual, hasEvidence, canVerifyContrast, occurrenceCount]);

	// Human checks land on their decision surface instead of generic fix prose.
	const defaultTab: TabId = needsVerification ? 'verify' : isManual ? 'review' : 'fix';

	const [tabState, setTabState] = useState<{ issueId: string; tab: TabId }>({
		issueId: issue.id,
		tab: defaultTab
	});
	const selectedTab = tabState.issueId === issue.id ? tabState.tab : defaultTab;
	const activeTab = availableTabs.includes(selectedTab) ? selectedTab : (availableTabs[0] ?? 'fix');

	const navigate = (delta: number) => {
		if (!onNavigate || currentIndex < 0) return;
		const next = currentIndex + delta;
		if (next < 0 || next >= totalCount) return;
		const target = issues[next];
		if (target) onNavigate(target.id);
	};

	const modalRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		const previousActiveElement = document.activeElement as HTMLElement | null;

		if (modalRef.current) {
			const closeBtn = modalRef.current.querySelector('.imodal__close') as HTMLElement | null;
			if (closeBtn) {
				closeBtn.focus();
			} else {
				const focusable = modalRef.current.querySelectorAll(
					'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
				);
				if (focusable.length > 0) {
					(focusable[0] as HTMLElement).focus();
				}
			}
		}

		return () => {
			if (previousActiveElement && typeof previousActiveElement.focus === 'function') {
				previousActiveElement.focus();
			}
		};
	}, []);

	useEffect(() => {
		const handleKeydown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') {
				event.preventDefault();
				onClose();
				return;
			}

			if (event.key === 'Tab') {
				if (modalRef.current) {
					const focusable = Array.from(
						modalRef.current.querySelectorAll(
							'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
						)
					).filter((el) => {
						const htmlEl = el as HTMLElement;
						return (
							htmlEl.offsetWidth > 0 ||
							htmlEl.offsetHeight > 0 ||
							htmlEl.getClientRects().length > 0
						);
					}) as HTMLElement[];

					if (focusable.length > 0) {
						const first = focusable[0];
						const last = focusable[focusable.length - 1];
						const active = document.activeElement as HTMLElement;

						if (event.shiftKey) {
							if (active === first || !focusable.includes(active)) {
								event.preventDefault();
								last.focus();
							}
						} else {
							if (active === last || !focusable.includes(active)) {
								event.preventDefault();
								first.focus();
							}
						}
					}
				}
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
	const severity = normalizeSeverity(issue.severity) ?? 'info';
	const moveTabFocus = (event: React.KeyboardEvent<HTMLButtonElement>, index: number) => {
		let nextIndex: number | null = null;
		if (event.key === 'ArrowRight') nextIndex = (index + 1) % availableTabs.length;
		if (event.key === 'ArrowLeft') nextIndex = (index - 1 + availableTabs.length) % availableTabs.length;
		if (event.key === 'Home') nextIndex = 0;
		if (event.key === 'End') nextIndex = availableTabs.length - 1;
		if (nextIndex === null) return;
		event.preventDefault();
		const nextTab = availableTabs[nextIndex];
		if (!nextTab) return;
		setTabState({ issueId: issue.id, tab: nextTab });
		event.currentTarget.parentElement
			?.querySelectorAll<HTMLButtonElement>('[role="tab"]')
			.item(nextIndex)
			.focus();
	};

	return (
		<div
			className="imodal__backdrop"
			role="dialog"
			aria-modal="true"
			aria-label="Issue details"
			onClick={onClose}
		>
			<div className="imodal" ref={modalRef} onClick={(e) => e.stopPropagation()}>
				{/* Compressed head: severity + status + context on one line, title below. */}
				<header className="imodal__head">
					<div className="imodal__head-top">
						<div className="imodal__chips">
							<span className={`imodal__sev sev-${severity}`}>
								<span className={getSeverityDotClass(severity)} aria-hidden="true" />
								{severity}
							</span>
							<span className="imodal__kind">{kindLabel}</span>
							{reviewVerdict ? (
								<span className={`imodal__verdict imodal__verdict--${reviewVerdict.verdict}`}>
									Reviewed · {reviewVerdict.verdict}
								</span>
							) : (
								needsReview && (
									<span className="imodal__verdict imodal__verdict--pending">Needs review</span>
								)
							)}
							{(pageLabel || occurrenceCount > 1) && (
								<span className="imodal__context">
									{pageLabel}
									{pageLabel && occurrenceCount > 1 ? ' · ' : ''}
									{occurrenceCount > 1 ? `${occurrenceCount} occurrences` : ''}
								</span>
							)}
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
										{currentIndex + 1} of {totalCount}
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
							{issue.scanner === 'lighthouse' ? 'Lighthouse' : issue.scanner} flagged this for human
							verification — no concrete DOM target was reported.
						</p>
					)}
				</header>

				<nav className="imodal__tabs" role="tablist">
					{availableTabs.map((tab, index) => (
						<button
							key={tab}
							type="button"
							role="tab"
							id={`issue-tab-${tab}`}
							aria-selected={activeTab === tab}
							aria-controls={`issue-panel-${tab}`}
							tabIndex={activeTab === tab ? 0 : -1}
							className="imodal__tab"
							onClick={() => setTabState({ issueId: issue.id, tab })}
							onKeyDown={(event) => moveTabFocus(event, index)}
						>
							{tab === 'review' && 'Review'}
							{tab === 'fix' && 'Fix'}
							{tab === 'evidence' && 'Evidence'}
							{tab === 'verify' && 'Verify'}
							{tab === 'occurrences' &&
								(occurrenceCount > 1 ? `Occurrences (${occurrenceCount})` : 'Occurrence')}
						</button>
					))}
				</nav>

				<div className="imodal__body">
					{activeTab === 'review' && (
						<div className="imodal__pane" id="issue-panel-review" role="tabpanel" aria-labelledby="issue-tab-review">
							<ManualReviewTab issue={issue} jobId={jobId} />
						</div>
					)}
					{activeTab === 'fix' && (
						<div className="imodal__pane" id="issue-panel-fix" role="tabpanel" aria-labelledby="issue-tab-fix">
							<section>
								<h3 className="imodal__pane-h">How to fix</h3>
								{contextualFix ? (
									<p className="imodal__fix">
										<ScannerText text={contextualFix} />
									</p>
								) : (
									<p className="imodal__fix imodal__fix--muted">
										No fix guidance available for this rule yet.
									</p>
								)}
							</section>
							<section>
								<h3 className="imodal__pane-h">About this rule</h3>
								<p className="imodal__desc">
									<ScannerText text={issue.description} />
								</p>
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
									Learn more about this rule ↗
								</a>
							)}
						</div>
					)}
					{activeTab === 'evidence' && (
						<div className="imodal__pane" id="issue-panel-evidence" role="tabpanel" aria-labelledby="issue-tab-evidence">
							<IssueEvidenceSection
								issue={issue}
								page={page}
								screenshotUrl={screenshotUrl}
								pageOverviewUrl={pageOverviewUrl}
							/>
						</div>
					)}
					{activeTab === 'verify' && (
						<div className="imodal__pane" id="issue-panel-verify" role="tabpanel" aria-labelledby="issue-tab-verify">
							<VerifyContrastTab
								issue={issue}
								page={page}
								pageOverviewUrl={pageOverviewUrl}
								jobId={jobId}
							/>
						</div>
					)}
					{activeTab === 'occurrences' && (
						<div className="imodal__pane" id="issue-panel-occurrences" role="tabpanel" aria-labelledby="issue-tab-occurrences">
							<h3 className="imodal__pane-h">
								{occurrenceCount === 1
									? 'Affected element'
									: `Affected elements (${occurrenceCount})`}
							</h3>
							<div className="imodal__occs">
								{(issue.occurrences ?? []).map((occ, idx) => (
									<IssueOccurrenceCard
										key={idx}
										occurrence={occ}
										index={idx}
										page={page}
										issueId={issue.id}
										pageOverviewUrl={pageOverviewUrl}
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
