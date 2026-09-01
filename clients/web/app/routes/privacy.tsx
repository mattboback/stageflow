import { Link, type MetaFunction } from 'react-router';

import { SiteFooter } from '../components/SiteFooter';
import { SiteHeader } from '../components/SiteHeader';
import { buildSiteMeta, GITHUB_URL, IS_HOSTED_DEMO, SITE_NAME } from '../lib/site-metadata';
import privacyStyles from './privacy.css?url';

export const links = () => [{ rel: 'stylesheet', href: privacyStyles }];

export const meta: MetaFunction = () =>
	buildSiteMeta({
		title: `${SITE_NAME} — Hosted demo data handling`,
		description: `${SITE_NAME} hosted-demo retention, delete, and cancel boundaries. Artifacts expire after 24 hours; durable job records are not erased by Delete.`,
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
							<li>Submitted static-site ZIP archives.</li>
							<li>Page URLs, titles, DOM snippets, response metadata, and scanner findings.</li>
							<li>Full-page screenshots and per-finding image evidence.</li>
							<li>Generated HTML and JSON reports.</li>
							<li>Scanner logs and timing/error information.</li>
							<li>Playwright storage state supplied for an authenticated scan.</li>
						</ul>
						<p>
							Do not submit confidential builds, private customer data, production credentials, or
							sensitive authenticated targets to the hosted demo.
						</p>
					</section>

					<section>
						<h2>Retention and access</h2>
						<p>
							The hosted demo expires staging uploads and ordinary completed scan artifacts after{' '}
							<strong>24 hours</strong>. Object-store deletion is asynchronous. Reports promoted as
							project baselines stay in private persistent storage until replaced or the project is
							deleted.
						</p>
						<p>
							The durable job record — submitted URL, scanner configuration, state, and timing — is
							not automatically deleted in this release. The 24-hour promise applies to uploaded
							files and ordinary generated object-store artifacts, not database records or promoted
							baselines.
						</p>
						<p>
							A job ID is an unguessable bearer-style reference: anyone with the job or report URL
							may retrieve its status and report until the data expires.
						</p>
					</section>

					<section>
						<h2>Delete and cancel</h2>
						<p>
							<strong>Delete this scan</strong> removes staging and artifact objects and hides the
							job from later reads. It does not erase the durable job record. Promoted API project
							baselines are not deleted by this action.
						</p>
						<p>
							<strong>Cancel scan</strong> stops an in-flight job and tears down scanner pods. Cancel
							does not delete artifacts; use Delete after the job has stopped.
						</p>
					</section>

					<section>
						<h2>Authentication data</h2>
						<p>
							The hosted browser form accepts literal credentials only to support throwaway demo
							accounts. Never enter a personal, reused, customer, or production password. Prefer a
							self-hosted stack for any real login.
						</p>
					</section>

					<p className="privacy__source">
						Canonical operator documentation lives in{' '}
						<a href={sourceHref}>docs/privacy.md</a>.{' '}
						<Link to="/">Back to the demo</Link>.
					</p>
				</div>
			</main>
			<SiteFooter />
		</>
	);
}
