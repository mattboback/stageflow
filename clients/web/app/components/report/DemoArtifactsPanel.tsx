import { GITHUB_URL } from '../../lib/site-metadata';
import { DEMO_REPORT_URL } from '../../lib/hooks/useDemoReport';

/*
 * The artifacts tab on /demo.
 *
 * ArtifactsView lists presigned object-store URLs off a live job. There is no
 * job here, and fabricating one to satisfy the shape would emit download links
 * that 404 in a portfolio piece -- which is why ReportView takes this panel as
 * a slot rather than taking a job. What it can offer instead is the one
 * artifact that genuinely exists: the exact JSON this page is rendering.
 */
export function DemoArtifactsPanel() {
	return (
		<div className="artifacts">
			<section className="artifacts__card">
				<header className="artifacts__head">
					<h3>Report data</h3>
				</header>
				<div className="artifacts__body">
					<ul className="artifacts__links">
						<li>
							<a href={DEMO_REPORT_URL} target="_blank" rel="noopener noreferrer">
								report.json ↗
							</a>
						</li>
						<li>
							<a
								href={`${GITHUB_URL}/blob/main/libs/contracts/report/schemas/unified-report.v2.schema.json`}
								target="_blank"
								rel="noopener noreferrer"
							>
								The schema it validates against ↗
							</a>
						</li>
					</ul>
					<p className="artifacts__note">
						This is the file this page renders, byte for byte — the same merged report a scan
						writes, from the CLI, from CI, or from the browser.
					</p>
				</div>
			</section>

			<section className="artifacts__card">
				<header className="artifacts__head">
					<h3>On a real scan</h3>
				</header>
				<div className="artifacts__body">
					<p className="artifacts__note">
						A live report lists signed download links here for each scanner&apos;s raw output, the
						aggregated HTML report, the stage log, and the scan recipe. Those come from the job that
						produced the report, and this page deliberately has no job behind it.
					</p>
					<ul className="artifacts__links">
						<li>
							<a href={`${GITHUB_URL}#readme`} target="_blank" rel="noopener noreferrer">
								Run one yourself ↗
							</a>
						</li>
					</ul>
				</div>
			</section>
		</div>
	);
}
