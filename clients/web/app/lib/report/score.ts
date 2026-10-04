import type { IssueDetail, IssueSeverity, UnifiedReport } from '../types/unified-report';
import { rawIssueId } from './occurrence-mode';
import { incompleteScanners } from './scanner-status';
import { tallySeverities, type SeverityLevel } from './severity';

/*
 * The site score, recomputed so the report can say what lowers it and what a
 * fix is worth. It mirrors calculateAccessibilityScore in
 * services/orchestrator/internal/aggregate/report_aggregator_utils.go. Both test
 * suites run libs/contracts/report/fixtures/score-cases.json, so neither side
 * can change the formula alone.
 */

const PENALTY: Record<IssueSeverity, number> = {
	critical: 20,
	serious: 6,
	moderate: 2,
	minor: 0.5,
	info: 0
};

export type ScoreCounts = Partial<Record<SeverityLevel, number>>;

export function computeSiteScore(counts: ScoreCounts, pagesScanned: number): number {
	// Summed in the orchestrator's order, so the floating-point result matches.
	let penalty =
		(counts.critical ?? 0) * PENALTY.critical +
		(counts.serious ?? 0) * PENALTY.serious +
		(counts.moderate ?? 0) * PENALTY.moderate +
		(counts.minor ?? 0) * PENALTY.minor;
	if (pagesScanned > 1) penalty /= pagesScanned;
	if (penalty <= 0) return 100;
	const scaled = Math.min(100, 20 * Math.log10(penalty + 1) + penalty * 0.25);
	return Math.max(0, Math.trunc(100 - scaled + 0.5));
}

/**
 * The score the report's dial shows, or null when it shows none: a
 * verification run, a partial one, or a report without a score. Anything
 * printed beside the dial, such as a baseline comparison, has to agree with it.
 */
export function shownSiteScore(report: UnifiedReport): number | null {
	if (report.meta.scanContext?.purpose === 'verification') return null;
	if (incompleteScanners(report.scanners).length > 0) return null;
	const score = report.assessment?.score ?? report.summary.score;
	return typeof score === 'number' ? Math.round(score) : null;
}

export interface ScoreModel {
	/** The score the report shows, which `computeSiteScore` reproduces from `issues`. */
	score: number;
	pagesScanned: number;
	/** The findings the score counts, as the report stores them (not split per occurrence). */
	issues: IssueDetail[];
}

/**
 * What the shown score counts, or null when the report's own number cannot be
 * reproduced: a verification run or a partial one (neither shows a score), or
 * a score that was not computed from these findings. A breakdown that did not
 * add up to the number beside it would be worse than none.
 */
export function explainScore(report: UnifiedReport): ScoreModel | null {
	const score = shownSiteScore(report);
	if (score === null) return null;
	const assessment = report.assessment?.score === undefined ? null : report.assessment;
	// The orchestrator's rules: the assessment counts actionable findings; an
	// older summary score counts everything but unconfirmed axe observations.
	const issues = report.issues.filter((issue) =>
		assessment
			? (assessment.dispositionByIssue[issue.id] ?? 'actionable') === 'actionable'
			: issue.scannerData?.axeIncomplete !== true
	);
	const pagesScanned = report.pages.length;
	if (computeSiteScore(tallySeverities(issues), pagesScanned) !== score) return null;
	return { score, pagesScanned, issues };
}

/**
 * The score once these findings are gone. Takes report or display (per
 * occurrence) ids. Recompute a set of fixes together rather than adding up
 * their gains: the curve flattens, so gains do not add.
 */
export function scoreIfFixed(model: ScoreModel, fixedIds: Iterable<string>): number {
	const fixed = new Set(Array.from(fixedIds, rawIssueId));
	return computeSiteScore(
		tallySeverities(model.issues.filter((issue) => !fixed.has(issue.id))),
		model.pagesScanned
	);
}

export interface ScannerPenaltyShare {
	scanner: string;
	share: number;
	/** The score with every finding from this scanner fixed. */
	scoreWithout: number;
}

/**
 * Each scanner's part of the severity-weighted penalty, largest first. These
 * are shares of what goes into the curve, not of points lost, which do not add.
 */
export function penaltyShareByScanner(model: ScoreModel): ScannerPenaltyShare[] {
	const penaltyByScanner = new Map<string, number>();
	let total = 0;
	for (const issue of model.issues) {
		const penalty = PENALTY[issue.severity];
		if (!penalty) continue;
		penaltyByScanner.set(issue.scanner, (penaltyByScanner.get(issue.scanner) ?? 0) + penalty);
		total += penalty;
	}
	return Array.from(penaltyByScanner, ([scanner, penalty]) => ({
		scanner,
		share: penalty / total,
		scoreWithout: scoreIfFixed(
			model,
			model.issues.filter((issue) => issue.scanner === scanner).map((issue) => issue.id)
		)
	})).sort((a, b) => b.share - a.share || a.scanner.localeCompare(b.scanner));
}

/**
 * Whole percents for shares of one total, adding up to 100 (largest
 * remainder), so a breakdown never reads 101%.
 */
export function sharePercents(shares: readonly number[]): number[] {
	const exact = shares.map((share) => share * 100);
	const percents = exact.map(Math.floor);
	let missing =
		Math.round(exact.reduce((sum, value) => sum + value, 0)) -
		percents.reduce((sum, value) => sum + value, 0);
	const byRemainder = exact
		.map((value, index) => ({ index, remainder: value - Math.floor(value) }))
		.sort((a, b) => b.remainder - a.remainder || a.index - b.index);
	for (const { index } of byRemainder) {
		if (missing <= 0) break;
		percents[index] = (percents[index] ?? 0) + 1;
		missing -= 1;
	}
	return percents;
}
