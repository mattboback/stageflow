import type { IssueOccurrence, PageSummary } from '../../lib/types/unified-report';

interface Props {
	occurrence: IssueOccurrence;
	index: number;
	page: PageSummary | null;
}

export function IssueOccurrenceCard({ occurrence, index, page }: Props) {
	const pageLabel = page?.path ?? page?.url ?? occurrence.pageId ?? null;

	return (
		<article className="occ">
			<header className="occ__head">
				<span className="occ__idx">#{index + 1}</span>
				{pageLabel && <span className="occ__page">{pageLabel}</span>}
				{occurrence.label && <span className="occ__label">{occurrence.label}</span>}
			</header>
			{occurrence.selector && (
				<dl className="occ__field">
					<dt>Selector</dt>
					<dd>
						<code>{occurrence.selector}</code>
					</dd>
				</dl>
			)}
			{occurrence.ancestorPath && (
				<dl className="occ__field">
					<dt>Ancestor path</dt>
					<dd>
						<code>{occurrence.ancestorPath}</code>
					</dd>
				</dl>
			)}
			{(occurrence.html || occurrence.contextHtml) && (
				<dl className="occ__field">
					<dt>HTML</dt>
					<dd>
						<pre>
							<code>{occurrence.html ?? occurrence.contextHtml}</code>
						</pre>
					</dd>
				</dl>
			)}
			{occurrence.failureSummary && (
				<dl className="occ__field">
					<dt>Why this fails</dt>
					<dd>{occurrence.failureSummary}</dd>
				</dl>
			)}
		</article>
	);
}
