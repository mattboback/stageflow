import { useMemo } from 'react';

import type { PageOverviewElement, PageSummary } from '../../lib/types/unified-report';
import { getCroppedViewBox, getSeverityStrokeColor } from '../../lib/report';

interface Props {
	page: PageSummary;
	overviewUrl: string;
	element: PageOverviewElement;
	ariaLabel: string;
	className?: string;
}

/*
 * Page-overview screenshot cropped tight around one flagged element, with a
 * severity-colored box on top. Shared by the modal's Evidence tab and the
 * per-occurrence cards.
 */
export function ElementCrop({ page, overviewUrl, element, ariaLabel, className }: Props) {
	const overview = page.pageOverview;
	const viewBox = useMemo(
		() =>
			overview
				? getCroppedViewBox(overview.pageWidth, overview.pageHeight, element)
				: null,
		[overview, element]
	);

	if (!overview || !viewBox) return null;

	return (
		<svg
			className={className ?? 'ecrop'}
			viewBox={`${viewBox.x} ${viewBox.y} ${viewBox.width} ${viewBox.height}`}
			preserveAspectRatio="xMidYMid meet"
			role="img"
			aria-label={ariaLabel}
		>
			<image
				href={overviewUrl}
				x={0}
				y={0}
				width={overview.pageWidth}
				height={overview.pageHeight}
			/>
			<rect
				x={element.x}
				y={element.y}
				width={element.width}
				height={element.height}
				fill="transparent"
				stroke={getSeverityStrokeColor(element.severity)}
				strokeWidth={3}
			/>
		</svg>
	);
}

