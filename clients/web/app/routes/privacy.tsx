import { Link, type MetaFunction } from 'react-router';

import { SiteFooter } from '../components/SiteFooter';
import { SiteHeader } from '../components/SiteHeader';
import { buildSiteMeta, GITHUB_URL, IS_HOSTED_DEMO, SITE_NAME } from '../lib/site-metadata';
import privacyStyles from './privacy.css?url';

export const links = () => [{ rel: 'stylesheet', href: privacyStyles }];

export const meta: MetaFunction = () =>
	buildSiteMeta({
		title: `${SITE_NAME} — Hosted demo data handling`,
		description: `${SITE_NAME} hosted-demo retention and delete boundaries. Scans and their files are deleted after 24 hours; database backups are kept for 14 days.`,
		path: '/privacy'
	});

export default function Privacy() {
	const sourceHref = `${GITHUB_URL}/blob/main/docs/privacy.md`;

	return (
		<>
			<SiteHeader />
			<main id="main" className="privacy">
				<div className="wrap wrap--app">
					<p className="eyebrow">Data handling</p>
					<h1>Hosted demo privacy</h1>
					<p className="lede">
						{IS_HOSTED_DEMO
							? 'This page describes the public no-account demo you are using. Self-hosted operators control their own storage, access, and retention.'
							: 'This page describes the public no-account demo at stageflow.org. This deployment is self-hosted; your operator controls storage, access, and retention.'}
					</p>

					<section>
						<h2>What a scan may store</h2>
						<p>Depending on the scan type and enabled scanners, StageFlow may temporarily store:</p>
						<ul>
							{!IS_HOSTED_DEMO && <li>Submitted static-site ZIP archives.</li>}
							<li>Page URLs, titles, DOM snippets, response metadata, and scanner findings.</li>
							<li>Full-page screenshots and per-finding image evidence.</li>
							<li>Generated HTML and JSON reports.</li>
							<li>Scanner logs and timing/error information.</li>
							<li>
								Playwright storage state or form-login values supplied for an authenticated scan.
							</li>
						</ul>
						<p>
							Do not submit confidential builds, private customer data, production credentials, or
							sensitive authenticated targets to the hosted demo.
						</p>
					</section>

					<section>
						<h2>Retention and access</h2>
						<p>
							A finished scan's job record and files are deleted <strong>24 hours</strong> after it
							completes. The job record holds the submitted URL, scanner configuration, state, and
							timing. Browser-local project reports and baseline files stay on your device until you
							remove them.
						</p>
						<p>
							Report files and screenshots are not backed up. The database is backed up nightly and
							daily backups are kept for 14 days, with all backups younger than 48 hours preserved.
							A job record can outlive the 24 hours in a backup.
						</p>
						<p>
							A job ID is an unguessable bearer-style reference: anyone with the job or report URL
							may retrieve its status and report until the data expires.
						</p>
					</section>

					<section>
						<h2>Delete</h2>
						<p>
							<strong>Delete this scan</strong> removes a finished scan's job record and files at
							once. It does not reach backups already taken. Browser-saved reports and local
							baseline files are not deleted by this action.
						</p>
					</section>

					<section>
						<h2>Authentication data</h2>
						<p>
							The hosted browser form accepts literal credentials only to support throwaway demo
							accounts. They are scrubbed from the job when it ends and removed from new hosted
							database backups. Older backups can contain form values from running scans until their
							14-day retention expires. Never enter a personal, reused, customer, or production
							password. Prefer a self-hosted stack for any real login.
						</p>
					</section>

					<p className="privacy__source">
						Canonical operator documentation lives in <a href={sourceHref}>docs/privacy.md</a>.{' '}
						<Link to="/">Back to the demo</Link>.
					</p>
				</div>
			</main>
			<SiteFooter />
		</>
	);
}
