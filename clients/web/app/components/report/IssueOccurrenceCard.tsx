import type { IssueOccurrence, PageSummary } from '../../lib/types/unified-report';

import { findOverviewElement } from '../../lib/report';

import { ElementCrop } from './ElementCrop';

interface Props {
	occurrence: IssueOccurrence;
	index: number;
	page: PageSummary | null;
	issueId?: string;
	pageOverviewUrl?: string | null;
}

export function IssueOccurrenceCard({ occurrence, index, page, issueId, pageOverviewUrl }: Props) {
	const pageLabel = page?.path ?? page?.url ?? occurrence.pageId ?? null;
	const overviewElement = issueId ? findOverviewElement(page, issueId, index) : null;

	/* The ancestor path only earns space when it says more than the selector. */
	const selector = occurrence.selector?.trim() ?? '';
	const ancestorPath = occurrence.ancestorPath?.trim() ?? '';
	const ancestorAddsContext =
		ancestorPath.length > 0 && ancestorPath !== selector && !selector.endsWith(ancestorPath);

	return (
		<article className="occ">
			<header className="occ__head">
				<span className="occ__idx">Occurrence {index + 1}</span>
				{pageLabel && <span className="occ__page">Page {pageLabel}</span>}
				{occurrence.label && <span className="occ__label">{occurrence.label}</span>}
			</header>
			{page && pageOverviewUrl && overviewElement && (
				<ElementCrop
					page={page}
					overviewUrl={pageOverviewUrl}
					element={overviewElement}
					ariaLabel={`Screenshot of occurrence ${index + 1}`}
					className="ecrop ecrop--sm"
				/>
			)}
			{occurrence.selector && (
				<dl className="occ__field">
					<dt>Selector</dt>
					<dd>
						<code className="code-well">{occurrence.selector}</code>
					</dd>
				</dl>
			)}
			{ancestorAddsContext && (
				<details className="occ__dom">
					<summary>DOM context</summary>
					<code className="code-well">{ancestorPath}</code>
				</details>
			)}
			{(occurrence.html || occurrence.contextHtml) && (
				<dl className="occ__field">
					<dt>HTML</dt>
					<dd>
						<pre className="code-well">
							<code>{occurrence.html ?? occurrence.contextHtml}</code>
						</pre>
					</dd>
				</dl>
			)}
		</article>
	);
}
