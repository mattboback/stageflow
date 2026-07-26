import { Link } from 'react-router';
import {
	DOCS_URL,
	GITHUB_URL,
	IS_HOSTED_DEMO,
	SITE_NAME,
	SITE_TAGLINE
} from '../lib/site-metadata';
import { BrandMark } from './BrandMark';

/* slim: meta row only — used under active application workflows where the
   sitemap is noise (scan progress, report). */
export function SiteFooter({ slim = false }: { slim?: boolean }) {
	const meta = (
		<div className="wrap site-footer__meta">
			<span>MIT licensed</span>
			<span>
				{IS_HOSTED_DEMO
					? 'Hosted demo — submitted targets and scan data are processed by this deployment'
					: 'Self-hosted — scan data is processed by your deployment'}
			</span>
			<a href={`${GITHUB_URL}/blob/main/docs/privacy.md`}>Privacy</a>
			<a href={GITHUB_URL}>Source on GitHub</a>
		</div>
	);

	if (slim) {
		return <footer className="site-footer site-footer--slim">{meta}</footer>;
	}

	return (
		<footer className="site-footer">
			<div className="wrap site-footer__inner">
				<div className="site-footer__brand">
					<Link className="brand" to="/">
						<BrandMark />
						<span className="brand__name">{SITE_NAME}</span>
					</Link>
					<p className="muted site-footer__tagline">{SITE_TAGLINE}.</p>
				</div>
				<div className="site-footer__cols">
					<nav aria-label="Product" className="site-footer__col">
						<span className="label">Product</span>
						<Link to="/projects">Projects</Link>
						<Link to="/playground">Configure scan</Link>
						<Link to="/#scanners">Scanners</Link>
						<a href={DOCS_URL}>Documentation</a>
					</nav>
					<nav aria-label="Project" className="site-footer__col">
						<span className="label">Project</span>
						<a href={GITHUB_URL}>GitHub</a>
						<a href={`${GITHUB_URL}/blob/main/CHANGELOG.md`}>Changelog</a>
						<a href={`${GITHUB_URL}/security/policy`}>Security</a>
					</nav>
				</div>
			</div>
			{meta}
		</footer>
	);
}
