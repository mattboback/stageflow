import type { ReportError } from '../../lib/types/unified-report';

interface Props {
	errors: ReportError[] | undefined;
	compact?: boolean;
}

export function ErrorsView({ errors, compact = false }: Props) {
	const list = errors ?? [];

	if (list.length === 0 && compact) {
		return (
			<div className="errview errview--compact" data-testid="errors-zero-compact">
				<span className="errview__check" aria-hidden="true">
					✓
				</span>
				<span className="errview__ok-label">No report errors</span>
				<span className="errview__ok-sub">All scanners completed without pipeline failures.</span>
			</div>
		);
	}

	return (
		<section className="errview__card">
			<header className="errview__head">
				<h3>Report errors</h3>
			</header>
			{list.length === 0 ? (
				<div className="errview__zero">
					<span className="errview__check" aria-hidden="true">
						✓
					</span>
					No errors reported.
				</div>
			) : (
				<ul className="errview__list">
					{list.map((err, idx) => (
						<li key={idx} className="errview__row">
							<p className="errview__msg">{err.message}</p>
							<p className="errview__meta">
								{err.code} · {err.scope}
								{err.scannerId && <> · Scanner: {err.scannerId}</>}
								{err.pageId && <> · Page: {err.pageId}</>}
							</p>
							<p className="errview__retry">{err.retryable ? 'Retryable' : 'Not retryable'}</p>
						</li>
					))}
				</ul>
			)}
		</section>
	);
}
