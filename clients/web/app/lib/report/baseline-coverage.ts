import type { UnifiedReport } from '../types/unified-report';
import { incompleteScanners } from './scanner-status';

/** A reviewed baseline may contain manual checks, but needs completed page coverage. */
export function baselineCoverageProblem(report: UnifiedReport): string | null {
	if (incompleteScanners(report.scanners).length > 0)
		return 'This scan has incomplete scanner coverage. Run a complete scan before accepting a baseline.';
	const context = report.meta.scanContext;
	// Older saved reports do not record page coverage. Their existing limited
	// comparison warning remains the compatibility boundary.
	if (!context) return null;
	if (
		context.expectedPages.length === 0 ||
		report.scanners.length === 0 ||
		context.expectedPages.some((id) => !report.pages.some((page) => page.id === id))
	)
		return 'This scan is missing expected pages. Run a complete scan before accepting a baseline.';
	if (
		report.scanners.some(
			(scanner) =>
				scanner.status !== 'success' ||
				context.expectedPages.some(
					(id) => !scanner.pages?.some((page) => page.pageId === id && page.status === 'success')
				)
		)
	)
		return 'This scan has missing or unsuccessful page checks. Run a complete scan before accepting a baseline.';
	return null;
}
