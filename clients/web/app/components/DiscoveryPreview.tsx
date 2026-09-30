import { useState } from 'react';
import type { DiscoverPagesResponse } from '../lib/api/client';
import './DiscoveryPreview.css';

interface DiscoveryPreviewProps {
	result: DiscoverPagesResponse;
	onConfirm: (urls: string[]) => void;
	onCancel: () => void;
	confirmLabel?: string;
	busy?: boolean;
}

export function DiscoveryPreview({
	result,
	onConfirm,
	onCancel,
	confirmLabel = 'Use selected pages',
	busy = false
}: DiscoveryPreviewProps) {
	const [selected, setSelected] = useState(() => new Set(result.urls));
	const complete = result.complete ?? !result.truncated;
	const warnings = result.warnings ?? [];
	return (
		<section className="discovery-preview" aria-label="Website discovery preview">
			<h3>Choose pages to scan</h3>
			<p>
				{result.urls.length} public HTML pages found. {selected.size} selected.
			</p>
			{!complete && (
				<p role="status">
					Discovery coverage is incomplete. This scan will cover only the pages you select.
				</p>
			)}
			{warnings.length > 0 && (
				<ul className="discovery-preview__warnings">
					{warnings.map((warning) => (
						<li key={warning}>{warning}</li>
					))}
				</ul>
			)}
			<div className="discovery-preview__list">
				{result.urls.map((url) => (
					<label key={url}>
						<input
							type="checkbox"
							checked={selected.has(url)}
							disabled={busy}
							onChange={(event) => {
								setSelected((previous) => {
									const next = new Set(previous);
									if (event.target.checked) next.add(url);
									else next.delete(url);
									return next;
								});
							}}
						/>{' '}
						<span>{url}</span>
					</label>
				))}
			</div>
			<div className="discovery-preview__actions">
				<button
					type="button"
					className="btn btn--primary"
					disabled={busy || selected.size === 0}
					onClick={() => onConfirm(result.urls.filter((url) => selected.has(url)))}
				>
					{busy ? 'Starting…' : confirmLabel}
				</button>
				<button type="button" className="btn btn--ghost" disabled={busy} onClick={onCancel}>
					Cancel discovery
				</button>
			</div>
		</section>
	);
}
