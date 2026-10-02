import type { ScanResult } from '../../lib/types/scan';
import { buildApiUrl } from '../../lib/api/utils';
import { scannerLabel } from '../../lib/report/scanner-identity';

interface Props {
	jobId: string;
	job: ScanResult | null;
	onRefreshArtifacts?: () => void;
}

function formatTimestamp(iso?: string | null): string | null {
	if (!iso) return null;
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return null;
	return d.toLocaleString();
}

export function ArtifactsView({ jobId, job, onRefreshArtifacts }: Props) {
	const aggregatedJsonUrl = jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/results`) : null;
	const updatedLabel = formatTimestamp(job?.updated_at);
	const scannerEntries = job?.artifacts?.scanner_artifacts
		? Object.entries(job.artifacts.scanner_artifacts)
		: [];

	return (
		<div className="artifacts">
			<section className="artifacts__card">
				<header className="artifacts__head">
					<h3>Aggregated report</h3>
					{onRefreshArtifacts && (
						<button type="button" className="btn btn--ghost btn--sm" onClick={onRefreshArtifacts}>
							Refresh links
						</button>
					)}
				</header>
				<div className="artifacts__body">
					{aggregatedJsonUrl ? (
						<ul className="artifacts__links">
							<li>
								<a href={aggregatedJsonUrl} target="_blank" rel="noopener noreferrer">
									Aggregated JSON ↗
								</a>
							</li>
						</ul>
					) : (
						<p className="blankslate">No aggregated report yet.</p>
					)}
					<p className="artifacts__note">
						Scan files are kept for 24 hours.
						{updatedLabel && <> Last updated {updatedLabel}.</>}
					</p>
				</div>
			</section>

			<section className="artifacts__card">
				<header className="artifacts__head">
					<h3>Scanner results</h3>
				</header>
				<div className="artifacts__body">
					{scannerEntries.length === 0 ? (
						<p className="blankslate">No scanner results available yet.</p>
					) : (
						<div className="artifacts__matrix-wrap">
							<table className="artifacts__matrix">
								<thead>
									<tr>
										<th scope="col">Scanner</th>
										<th scope="col">JSON Results</th>
									</tr>
								</thead>
								<tbody>
									{scannerEntries.map(([scannerId, item]) => (
										<tr key={scannerId}>
											<th scope="row" className="artifacts__matrix-scanner">
												{scannerLabel(scannerId)}
											</th>
											<td>
												{item.results_json ? (
													<a href={item.results_json} target="_blank" rel="noopener noreferrer">
														JSON ↗
													</a>
												) : (
													<span className="muted">—</span>
												)}
											</td>
										</tr>
									))}
								</tbody>
							</table>
						</div>
					)}
				</div>
			</section>
		</div>
	);
}
