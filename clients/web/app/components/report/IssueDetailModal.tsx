import { useEffect, useMemo, useState, useRef } from 'react';
import { ChevronLeft, ChevronRight, X } from 'lucide-react';
import { useRovingTabList } from '../../lib/hooks/useRovingTabList';
import { useScrollLock } from '../../lib/hooks/useScrollLock';

import type { IssueDetail, PageSummary } from '../../lib/types/unified-report';
import type { ScreenshotArtifact } from '../../lib/types/scan';
import { cycleTabFocus, focusableWithin } from '../../lib/utils/focus-trap';
import {
	generateContextualFix,
	getIssueKind,
	getIssueKindLabel,
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

import { Pill } from '../Pill';
import { GuidanceText } from './GuidanceText';
import { ReviewNotesPanel } from './ReviewNotesPanel';
import { TruncatedEvidence } from './TruncatedEvidence';
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
	onVerify?: () => void;
	verificationUnavailable?: string | null;
}

type TabId = 'review' | 'fix' | 'evidence' | 'verify' | 'occurrences';

/* The contract's UserGroup tokens, as the people they stand for. */
const USER_GROUP_LABELS: Record<string, string> = {
	blind: 'Blind users',
	'low-vision': 'Low-vision users',
	motor: 'Users with motor impairments',
	cognitive: 'Users with cognitive disabilities',
	deaf: 'Deaf and hard-of-hearing users',
	vestibular: 'Users with vestibular disorders',
	all: 'All users'
};

function userGroupLabel(group: string): string {
	const words = group.replaceAll('-', ' ');
	return USER_GROUP_LABELS[group] ?? words.charAt(0).toUpperCase() + words.slice(1);
}

export function IssueDetailModal({
	issue,
	jobId = '',
	page,
	issues,
	screenshots,
	onClose,
	onNavigate,
	onVerify,
	verificationUnavailable
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
		hasOverviewCrop ||
		primaryOccurrence?.selector ||
		primaryOccurrence?.ancestorPath ||
		primaryOccurrence?.contextHtml ||
		primaryOccurrence?.html ||
		primaryOccurrence?.failureSummary ||
		(issue.scanner === 'lighthouse' && issue.scannerData?.details)
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

	useScrollLock();

	useEffect(() => {
		// Both layouts start a newly selected finding/tab at its content, even
		// when the previous phone pane was scrolled down to the review notes.
		modalRef.current
			?.querySelectorAll<HTMLElement>('.imodal__content, .imodal__body')
			.forEach((region) => {
				region.scrollTop = 0;
			});
	}, [issue.id, activeTab]);

	useEffect(() => {
		const previousActiveElement = document.activeElement as HTMLElement | null;

		if (modalRef.current) {
			// Prefer the close button so Escape and Tab both start from a known place;
			// otherwise fall back to whatever the trap considers first.
			const closeBtn = modalRef.current.querySelector<HTMLElement>('.imodal__close');
			const target = closeBtn ?? focusableWithin(modalRef.current)[0];
			target?.focus();
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
					cycleTabFocus(focusableWithin(modalRef.current), event);
				}
				return;
			}

			// Child widgets own keys they have already handled (tabs, color sampler).
			if (event.defaultPrevented) return;

			const target = event.target as HTMLElement | null;
			const tag = target?.tagName;
			if (tag === 'INPUT' || tag === 'TEXTAREA' || target?.isContentEditable) return;
			if (event.metaKey || event.ctrlKey || event.altKey) return;

			if (event.key === 'p' || event.key === 'P') {
				modalRef.current?.querySelector<HTMLElement>('[data-shortcut="P"]')?.click();
				return;
			}
			if (event.key === 'f' || event.key === 'F') {
				modalRef.current?.querySelector<HTMLElement>('[data-shortcut="F"]')?.click();
				return;
			}
			if (event.key === 'j' || event.key === 'ArrowDown' || event.key === 'ArrowRight') {
				if (hasNext) {
					event.preventDefault();
					navigate(1);
				}
			} else if (event.key === 'k' || event.key === 'ArrowUp' || event.key === 'ArrowLeft') {
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
	const moveTabFocus = useRovingTabList(availableTabs.length, (index) => {
		const nextTab = availableTabs[index];
		if (nextTab) setTabState({ issueId: issue.id, tab: nextTab });
	});

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
							{reviewVerdict?.verdict ? (
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
										className="btn btn--icon"
										onClick={() => navigate(-1)}
										disabled={!hasPrev}
										aria-label="Previous issue (k)"
										title="Previous (k)"
									>
										<ChevronLeft size={18} aria-hidden="true" />
									</button>
									<span className="imodal__nav-count">
										{currentIndex + 1} of {totalCount}
									</span>
									<button
										type="button"
										className="btn btn--icon"
										onClick={() => navigate(1)}
										disabled={!hasNext}
										aria-label="Next issue (j)"
										title="Next (j)"
									>
										<ChevronRight size={18} aria-hidden="true" />
									</button>
								</>
							)}
							<button
								type="button"
								className="btn btn--icon imodal__close"
								onClick={onClose}
								aria-label="Close modal"
							>
								<X size={18} aria-hidden="true" />
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
					{onVerify && canVerifyContrast && (
						<div>
							<button
								type="button"
								className="btn btn--ghost btn--sm"
								disabled={Boolean(verificationUnavailable)}
								onClick={onVerify}
							>
								Verify this fix
							</button>
							{verificationUnavailable && <p className="imodal__sub">{verificationUnavailable}</p>}
						</div>
					)}
				</header>

				<nav className="tabs imodal__tabs" role="tablist">
					{availableTabs.map((tab, index) => (
						<button
							key={tab}
							type="button"
							role="tab"
							id={`issue-tab-${tab}`}
							aria-selected={activeTab === tab}
							aria-controls={`issue-panel-${tab}`}
							tabIndex={activeTab === tab ? 0 : -1}
							className="tab"
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

				<div className="imodal__content">
					<div className="imodal__body">
						{activeTab === 'review' && (
							<div
								className="imodal__pane"
								id="issue-panel-review"
								role="tabpanel"
								aria-labelledby="issue-tab-review"
							>
								<ManualReviewTab issue={issue} jobId={jobId} />
							</div>
						)}
						{activeTab === 'fix' && (
							<div
								className="imodal__pane"
								id="issue-panel-fix"
								role="tabpanel"
								aria-labelledby="issue-tab-fix"
							>
								<dl className="imodal__meta-bar" aria-label="Rule metadata">
									<div className="imodal__meta-cell">
										<dt>Rule &amp; scanner</dt>
										<dd>
											{issue.scanner} · <span className="mono">{issue.ruleId}</span>
										</dd>
									</div>
									<div className="imodal__meta-cell">
										<dt>WCAG standard</dt>
										<dd>
											{issue.wcagTags && issue.wcagTags.length > 0 ? (
												issue.wcagTags.join(' · ')
											) : (
												<span className="imodal__meta-muted">Not WCAG-mapped</span>
											)}
											{issue.helpUrl && (
												<>
													{' '}
													<a href={issue.helpUrl} target="_blank" rel="noopener noreferrer">
														Docs ↗
													</a>
												</>
											)}
										</dd>
									</div>
									<div className="imodal__meta-cell">
										<dt>Affected users</dt>
										<dd>
											{issue.userImpact?.affectedGroups &&
											issue.userImpact.affectedGroups.length > 0
												? issue.userImpact.affectedGroups.map(userGroupLabel).join(', ')
												: 'All users'}
										</dd>
									</div>
								</dl>
								<section className="imodal__fix-callout">
									<h3 className="imodal__pane-h">How to fix</h3>
									{contextualFix ? (
										<GuidanceText text={contextualFix} className="imodal__fix" />
									) : (
										<p className="imodal__fix imodal__fix--muted">
											No fix guidance available for this rule yet.
										</p>
									)}
								</section>
								{issue.userImpact &&
									(issue.userImpact.statement ||
										issue.userImpact.affectedGroups?.length ||
										issue.userImpact.userStory) && (
										<section>
											<h3 className="imodal__pane-h">Who this affects</h3>
											{issue.userImpact.statement && (
												<p className="imodal__desc">
													<ScannerText text={issue.userImpact.statement} />
												</p>
											)}
											{issue.userImpact.affectedGroups &&
												issue.userImpact.affectedGroups.length > 0 && (
													<ul className="imodal__groups" aria-label="Affected groups">
														{issue.userImpact.affectedGroups.map((group) => (
															<li key={group}>
																<Pill led={false}>{userGroupLabel(group)}</Pill>
															</li>
														))}
													</ul>
												)}
											{issue.userImpact.userStory && (
												<blockquote className="imodal__story">
													<ScannerText text={issue.userImpact.userStory} />
												</blockquote>
											)}
										</section>
									)}
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
							<div
								className="imodal__pane"
								id="issue-panel-evidence"
								role="tabpanel"
								aria-labelledby="issue-tab-evidence"
							>
								<IssueEvidenceSection issue={issue} page={page} pageOverviewUrl={pageOverviewUrl} />
							</div>
						)}
						{activeTab === 'verify' && (
							<div
								className="imodal__pane"
								id="issue-panel-verify"
								role="tabpanel"
								aria-labelledby="issue-tab-verify"
							>
								<VerifyContrastTab
									issue={issue}
									page={page}
									pageOverviewUrl={pageOverviewUrl}
									jobId={jobId}
								/>
							</div>
						)}
						{activeTab === 'occurrences' && (
							<div
								className="imodal__pane"
								id="issue-panel-occurrences"
								role="tabpanel"
								aria-labelledby="issue-tab-occurrences"
							>
								<h3 className="imodal__pane-h">
									{issueKind !== 'element'
										? 'Reported locations'
										: occurrenceCount === 1
											? 'Affected element'
											: `Affected elements (${occurrenceCount})`}
								</h3>
								<TruncatedEvidence issue={issue} />
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
					<footer className="imodal__foot">
						<ReviewNotesPanel key={issue.id} jobId={jobId} issueId={issue.id} />
					</footer>
				</div>
			</div>
		</div>
	);
}
