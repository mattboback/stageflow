export function getNextFocusIndex(
	length: number,
	activeIndex: number,
	shiftKey: boolean,
): number {
	if (length <= 0) return -1;
	if (activeIndex < 0 || activeIndex >= length) {
		return shiftKey ? length - 1 : 0;
	}

	if (shiftKey) {
		return activeIndex === 0 ? length - 1 : activeIndex - 1;
	}
	return activeIndex === length - 1 ? 0 : activeIndex + 1;
}
