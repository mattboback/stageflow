import type { IssueDetail, PageSummary } from '../../lib/types/unified-report';

import { findOverviewElement } from '../../lib/report';

import { ElementCrop } from './ElementCrop';

interface Props {
	issue: IssueDetail;
	page: PageSummary | null;
	screenshotUrl: string | null;
	pageOverviewUrl: string | null;
}

export function IssueEvidenceSection({ issue, page, screenshotUrl, pageOverviewUrl }: Props) {
	const primary = issue.occurrences?.[0] ?? null;
	const selector = primary?.selector ?? null;
	const html = primary?.html ?? primary?.contextHtml ?? null;
	const failureSummary = primary?.failureSummary ?? null;
	const pageLabel = page?.path ?? page?.url ?? issue.pageUrl ?? null;

	const overviewElement = findOverviewElement(page, issue.id);
	const canCrop = Boolean(page && pageOverviewUrl && overviewElement);

	const hasAny =
		canCrop || selector || html || failureSummary || screenshotUrl || pageLabel;

	if (!hasAny) {
		return (
			<p className="evidence__empty">
				No DOM evidence was captured for this finding.
			</p>
		);
	}

	return (
		<div className="evidence">
			{canCrop && page && pageOverviewUrl && overviewElement ? (
				<figure className="evidence__shot">
					<ElementCrop
						page={page}
						overviewUrl={pageOverviewUrl}
						element={overviewElement}
						ariaLabel={`Screenshot of the flagged element for ${issue.title}`}
					/>
					<figcaption>Element in context</figcaption>
				</figure>
			) : screenshotUrl ? (
				<figure className="evidence__shot">
					<img src={screenshotUrl} alt={`Screenshot evidence for ${issue.title}`} />
					<figcaption>Scanner-captured screenshot</figcaption>
				</figure>
			) : null}
			{pageLabel && (
				<dl className="evidence__field">
					<dt>Page</dt>
					<dd>{pageLabel}</dd>
				</dl>
			)}
			{selector && (
				<dl className="evidence__field">
					<dt>Selector</dt>
					<dd>
						<code>{selector}</code>
					</dd>
				</dl>
			)}
			{html && (
				<dl className="evidence__field">
					<dt>HTML</dt>
					<dd>
						<pre>
							<code>{html}</code>
						</pre>
					</dd>
				</dl>
			)}
			{failureSummary && (
				<dl className="evidence__field">
					<dt>Failure summary</dt>
					<dd>{failureSummary}</dd>
				</dl>
			)}
		</div>
	);
}
