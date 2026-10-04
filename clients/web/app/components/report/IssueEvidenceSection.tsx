import { useState } from 'react';
import { Check, Copy } from 'lucide-react';

import type { IssueDetail, PageSummary } from '../../lib/types/unified-report';

import { findOverviewElement, getIssueKind } from '../../lib/report';

import { GuidanceText } from './GuidanceText';
import { TruncatedEvidence } from './TruncatedEvidence';
import { LighthouseDetails } from './LighthouseDetails';
import { ElementCrop } from './ElementCrop';

interface Props {
	issue: IssueDetail;
	page: PageSummary | null;
	pageOverviewUrl: string | null;
}

function CopyButton({ value, label }: { value: string; label: string }) {
	const [copied, setCopied] = useState(false);
	return (
		<button
			type="button"
			className="codebox__copy"
			aria-label={label}
			onClick={() => {
				// writeText rejects when the document is not focused or clipboard
				// permission is denied. Swallowing that leaves the button silently
				// stuck on "Copy", which is the honest signal: nothing was copied.
				void navigator.clipboard
					?.writeText(value)
					.then(() => {
						setCopied(true);
						window.setTimeout(() => setCopied(false), 1600);
					})
					.catch(() => {
						setCopied(false);
					});
			}}
		>
			{copied ? <Check size={14} aria-hidden="true" /> : <Copy size={14} aria-hidden="true" />}
			{copied ? 'Copied' : 'Copy'}
		</button>
	);
}

export function IssueEvidenceSection({ issue, page, pageOverviewUrl }: Props) {
	const primary = issue.occurrences?.[0] ?? null;
	const selector = primary?.selector ?? null;
	const html = primary?.html ?? primary?.contextHtml ?? null;
	const failureSummary = primary?.failureSummary?.trim() || null;
	const pageLabel = page?.path ?? page?.url ?? issue.pageUrl ?? null;

	const overviewElement = findOverviewElement(page, issue.id);
	const canCrop = Boolean(page && pageOverviewUrl && overviewElement);

	/* Evidence order follows the rule kind: element findings are visual, so
	   the crop leads; document/page findings are structural, so markup leads. */
	const kind = getIssueKind(issue);
	const screenshotLeads = kind === 'element' && canCrop;

	const hasAny =
		canCrop ||
		selector ||
		html ||
		failureSummary ||
		pageLabel ||
		(issue.scanner === 'lighthouse' && issue.scannerData?.details);

	if (!hasAny) {
		return <p className="evidence__empty">No DOM evidence was captured for this finding.</p>;
	}

	const shot =
		canCrop && page && pageOverviewUrl && overviewElement ? (
			<figure className="evidence__shot">
				<ElementCrop
					page={page}
					overviewUrl={pageOverviewUrl}
					element={overviewElement}
					ariaLabel={`Screenshot of the flagged element for ${issue.title}`}
				/>
				<figcaption>
					Element in context ·{' '}
					<a href={pageOverviewUrl} target="_blank" rel="noopener noreferrer">
						Full screenshot ↗
					</a>
				</figcaption>
			</figure>
		) : null;

	return (
		<div className="evidence">
			{failureSummary && (
				<section>
					<h3 className="imodal__pane-h">What failed</h3>
					<GuidanceText text={failureSummary} className="evidence__summary" />
				</section>
			)}

			{screenshotLeads && shot}

			<dl className="evidence__meta">
				<div className="evidence__field evidence__field--inline">
					{pageLabel && (
						<>
							<dt>Page</dt>
							<dd>{pageLabel}</dd>
						</>
					)}
					<dt>Rule</dt>
					<dd>
						{issue.scanner} · <span className="mono">{issue.ruleId}</span>
					</dd>
				</div>
				{selector && (
					<div className="evidence__field">
						<dt>Selector</dt>
						<dd>
							<div className="codebox">
								<code className="code-well">{selector}</code>
								<CopyButton value={selector} label="Copy selector" />
							</div>
						</dd>
					</div>
				)}
				{html && (
					<div className="evidence__field">
						<dt>HTML</dt>
						<dd>
							<div className="codebox">
								<pre className="code-well">
									<code>{html}</code>
								</pre>
								<CopyButton value={html} label="Copy HTML" />
							</div>
						</dd>
					</div>
				)}
			</dl>

			{issue.scanner === 'lighthouse' && <LighthouseDetails details={issue.scannerData?.details} />}
			<TruncatedEvidence issue={issue} />
			{!screenshotLeads && shot}
		</div>
	);
}
