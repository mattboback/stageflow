import type { IssueDetail, PageSummary } from '../../lib/types/unified-report';

interface Props {
	issue: IssueDetail;
	page: PageSummary | null;
	screenshotUrl: string | null;
}

export function IssueEvidenceSection({ issue, page, screenshotUrl }: Props) {
	const primary = issue.occurrences?.[0] ?? null;
	const selector = primary?.selector ?? null;
	const html = primary?.html ?? primary?.contextHtml ?? null;
	const failureSummary = primary?.failureSummary ?? null;
	const pageLabel = page?.path ?? page?.url ?? issue.pageUrl ?? null;

	const hasAny = selector || html || failureSummary || screenshotUrl || pageLabel;

	if (!hasAny) {
		return (
			<p className="evidence__empty">
				No DOM evidence was captured for this finding.
			</p>
		);
	}

	return (
		<div className="evidence">
			{screenshotUrl && (
				<figure className="evidence__shot">
					<img src={screenshotUrl} alt={`Screenshot evidence for ${issue.title}`} />
				</figure>
			)}
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
