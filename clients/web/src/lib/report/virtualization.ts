export interface VirtualizationParams {
	scrollTop: number;
	viewportHeight: number;
	rowHeight: number;
	totalItems: number;
	overScan?: number;
}

export interface VirtualWindow {
	startIndex: number;
	endIndex: number;
	offset: number;
	visibleCount: number;
}

export function getVirtualWindow({
	scrollTop,
	viewportHeight,
	rowHeight,
	totalItems,
	overScan = 6,
}: VirtualizationParams): VirtualWindow {
	if (totalItems <= 0 || rowHeight <= 0 || viewportHeight <= 0) {
		return { startIndex: 0, endIndex: 0, offset: 0, visibleCount: 0 };
	}

	const start = Math.max(0, Math.floor(scrollTop / rowHeight) - overScan);
	const visible = Math.ceil(viewportHeight / rowHeight) + overScan * 2;
	const end = Math.min(totalItems, start + visible);
	return {
		startIndex: start,
		endIndex: end,
		offset: start * rowHeight,
		visibleCount: visible,
	};
}
