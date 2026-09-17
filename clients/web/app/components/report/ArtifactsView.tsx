import type { ScanResult } from '../../lib/types/scan';
import { buildApiUrl } from '../../lib/api/utils';
import { scannerLabel } from '../../lib/report/scanner-identity';

interface Props {
	jobId: string;
	job: ScanResult | null;
	onRefreshArtifacts?: () => void;
}

interface Link {
	href: string;
	label: string;
}

function formatTimestamp(iso?: string | null): string | null {
	if (!iso) return null;
	const d = new Date(iso);
	if (Number.isNaN(d.getTime())) return null;
	return d.toLocaleString();
}

export function ArtifactsView({ jobId, job, onRefreshArtifacts }: Props) {
	const aggregatedJsonUrl = jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/results`) : null;
	const aggregatedHtmlUrl = jobId ? buildApiUrl(`/api/v1/jobs/${jobId}/report`) : null;
	const artifacts = job?.artifacts ?? null;
	const updatedLabel = formatTimestamp(job?.updated_at);

	const aggregatedLinks: Link[] = [];
	if (aggregatedJsonUrl)
		aggregatedLinks.push({ href: aggregatedJsonUrl, label: 'Aggregated JSON' });
	if (aggregatedHtmlUrl)
		aggregatedLinks.push({ href: aggregatedHtmlUrl, label: 'Primary HTML report' });
	if (artifacts?.scan_stage_log)
		aggregatedLinks.push({ href: artifacts.scan_stage_log, label: 'Scan stage log' });
	if (artifacts?.scan_recipe)
		aggregatedLinks.push({ href: artifacts.scan_recipe, label: 'Scan recipe' });
	if (artifacts?.extraction_stage_log)
		aggregatedLinks.push({ href: artifacts.extraction_stage_log, label: 'Extraction log' });
	if (artifacts?.extraction_recipe)
		aggregatedLinks.push({ href: artifacts.extraction_recipe, label: 'Extraction recipe' });

	const scannerEntries = artifacts?.scanner_artifacts
		? Object.entries(artifacts.scanner_artifacts)
		: [];

	return (
		<div className="artifacts">
			<section className="artifacts__card">
				<header className="artifacts__head">
					<h3>Aggregated report links</h3>
					{onRefreshArtifacts && (
						<button type="button" className="btn btn--ghost btn--sm" onClick={onRefreshArtifacts}>
							Refresh links
						</button>
					)}
				</header>
				<div className="artifacts__body">
					{aggregatedLinks.length === 0 ? (
						<p className="blankslate">No aggregated artifacts yet.</p>
					) : (
						<ul className="artifacts__links">
							{aggregatedLinks.map((link) => (
								<li key={link.label}>
									<a href={link.href} target="_blank" rel="noopener noreferrer">
										{link.label} ↗
									</a>
								</li>
							))}
						</ul>
					)}
					<p className="artifacts__note">
						Links are signed and can expire. Refresh to regenerate if needed.
						{updatedLabel && <> Last updated {updatedLabel}.</>}
					</p>
				</div>
			</section>

			<section className="artifacts__card">
				<header className="artifacts__head">
					<h3>Scanner artifacts</h3>
				</header>
				<div className="artifacts__body">
					{scannerEntries.length === 0 ? (
						<p className="blankslate">No scanner artifacts available yet.</p>
					) : (
						<div className="artifacts__matrix-wrap">
							<table className="artifacts__matrix">
								<thead>
									<tr>
										<th scope="col">Scanner</th>
										<th scope="col">JSON Results</th>
										<th scope="col">HTML Report</th>
										<th scope="col">Stage Log</th>
										<th scope="col">Scan Recipe</th>
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
											<td>
												{item.report_html ? (
													<a href={item.report_html} target="_blank" rel="noopener noreferrer">
														HTML ↗
													</a>
												) : (
													<span className="muted">—</span>
												)}
											</td>
											<td>
												{item.scan_stage_log ? (
													<a href={item.scan_stage_log} target="_blank" rel="noopener noreferrer">
														Log ↗
													</a>
												) : (
													<span className="muted">—</span>
												)}
											</td>
											<td>
												{item.scan_recipe ? (
													<a href={item.scan_recipe} target="_blank" rel="noopener noreferrer">
														Recipe ↗
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
