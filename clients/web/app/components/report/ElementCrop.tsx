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
	const isDocumentLevel = overview
		? element.width >= overview.pageWidth * 0.9 && element.height >= overview.pageHeight * 0.9
		: false;
	const viewBox = useMemo(() => {
		if (!overview) return null;
		if (isDocumentLevel) {
			const height = Math.min(overview.pageHeight, overview.pageWidth * 0.5625);
			return { x: 0, y: 0, width: overview.pageWidth, height };
		}
		return getCroppedViewBox(overview.pageWidth, overview.pageHeight, element);
	}, [overview, element, isDocumentLevel]);

	if (!overview || !viewBox) return null;

	const svg = (
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
				strokeWidth={2}
				// Screen pixels, so the box reads the same in a thumbnail and a full crop.
				vectorEffect="non-scaling-stroke"
			/>
		</svg>
	);

	if (!isDocumentLevel) return svg;
	return (
		<span className="ecrop-wrap">
			{svg}
			<span className="ecrop-badge">Document-level finding — showing top viewport</span>
		</span>
	);
}
