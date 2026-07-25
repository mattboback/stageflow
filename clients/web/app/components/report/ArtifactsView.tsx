import type { ScanResult } from '../../lib/types/scan';
import { buildApiUrl } from '../../lib/api/utils';

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
						<button type="button" className="artifacts__refresh" onClick={onRefreshArtifacts}>
							Refresh links
						</button>
					)}
				</header>
				<div className="artifacts__body">
					{aggregatedLinks.length === 0 ? (
						<p className="artifacts__empty">No aggregated artifacts yet.</p>
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
						<p className="artifacts__empty">No scanner artifacts available yet.</p>
					) : (
						<ul className="artifacts__scanners">
							{scannerEntries.map(([scannerId, item]) => {
								const links: Link[] = [];
								if (item.results_json)
									links.push({ href: item.results_json, label: 'JSON results' });
								if (item.report_html) links.push({ href: item.report_html, label: 'HTML report' });
								if (item.scan_stage_log)
									links.push({ href: item.scan_stage_log, label: 'Stage log' });
								if (item.scan_recipe) links.push({ href: item.scan_recipe, label: 'Scan recipe' });
								return (
									<li key={scannerId}>
										<p className="artifacts__scanner-name">{scannerId}</p>
										<ul className="artifacts__links artifacts__links--inline">
											{links.map((link) => (
												<li key={link.label}>
													<a href={link.href} target="_blank" rel="noopener noreferrer">
														{link.label} ↗
													</a>
												</li>
											))}
										</ul>
									</li>
								);
							})}
						</ul>
					)}
				</div>
			</section>
		</div>
	);
}
