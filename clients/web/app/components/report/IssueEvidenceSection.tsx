import { useState } from 'react';
import { Check, Copy } from 'lucide-react';

import type { IssueDetail, PageSummary } from '../../lib/types/unified-report';

import { findOverviewElement } from '../../lib/report';

import { ElementCrop } from './ElementCrop';

interface Props {
	issue: IssueDetail;
	page: PageSummary | null;
	screenshotUrl: string | null;
	pageOverviewUrl: string | null;
}

function CopyButton({ value, label }: { value: string; label: string }) {
	const [copied, setCopied] = useState(false);
	return (
		<button
			type="button"
			className="evidence__copy"
			aria-label={label}
			onClick={() => {
				navigator.clipboard?.writeText(value).then(() => {
					setCopied(true);
					window.setTimeout(() => setCopied(false), 1600);
				});
			}}
		>
			{copied ? <Check size={14} aria-hidden="true" /> : <Copy size={14} aria-hidden="true" />}
			{copied ? 'Copied' : 'Copy'}
		</button>
	);
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
			{failureSummary && (
				<section>
					<h3 className="imodal__pane-h">What failed</h3>
					<p className="evidence__summary">{failureSummary}</p>
				</section>
			)}

			<dl className="evidence__meta">
				{pageLabel && (
					<div className="evidence__field">
						<dt>Page</dt>
						<dd>{pageLabel}</dd>
					</div>
				)}
				<div className="evidence__field">
					<dt>Rule</dt>
					<dd>
						{issue.scanner} · <span className="mono">{issue.ruleId}</span>
					</dd>
				</div>
				{selector && (
					<div className="evidence__field">
						<dt>
							Selector <CopyButton value={selector} label="Copy selector" />
						</dt>
						<dd>
							<code className="code-well">{selector}</code>
						</dd>
					</div>
				)}
				{html && (
					<div className="evidence__field">
						<dt>
							HTML <CopyButton value={html} label="Copy HTML" />
						</dt>
						<dd>
							<pre className="code-well">
								<code>{html}</code>
							</pre>
						</dd>
					</div>
				)}
			</dl>

			{canCrop && page && pageOverviewUrl && overviewElement ? (
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
			) : screenshotUrl ? (
				<figure className="evidence__shot">
					<img src={screenshotUrl} alt={`Screenshot evidence for ${issue.title}`} />
					<figcaption>
						Scanner-captured screenshot ·{' '}
						<a href={screenshotUrl} target="_blank" rel="noopener noreferrer">
							Full screenshot ↗
						</a>
					</figcaption>
				</figure>
			) : null}
		</div>
	);
}
