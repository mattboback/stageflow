import type { ScanResult } from '$lib/types/scan';

import type { SSEUpdate } from './types';

export function applySseUpdate(result: ScanResult, update: SSEUpdate): ScanResult {
	const newProgress = update.progress
		? (() => {
				const currentPage = update.progress.currentPage;
				const totalPages = update.progress.totalPages;
				const rawPercentage = totalPages > 0 ? (currentPage / totalPages) * 100 : 0;
				return {
					current_page: currentPage,
					total_pages: totalPages,
					percentage: Math.max(0, Math.min(100, Math.round(rawPercentage)))
				};
			})()
		: result.progress;

	return {
		...result,
		state: update.state,
		...(newProgress !== undefined ? { progress: newProgress } : {}),
		...(update.error !== undefined ? { error: update.error } : {}),
		...(update.error_details !== undefined ? { error_details: update.error_details } : {}),
		...(update.stage !== undefined ? { last_stage: update.stage } : {})
	};
}
