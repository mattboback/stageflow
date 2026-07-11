import { Link } from 'react-router';

const GITHUB_URL = 'https://github.com/mattboback/stageflow';
const DOCS_URL = `${GITHUB_URL}/tree/main/docs`;

/* slim: meta row only — used under active application workflows where the
   sitemap is noise (scan progress, report). */
export function SiteFooter({ slim = false }: { slim?: boolean }) {
	const meta = (
		<div className="wrap site-footer__meta">
			<span>MIT licensed</span>
			<span>Runs in your infrastructure — your data stays with you</span>
			<a href={GITHUB_URL}>github.com/mattboback/stageflow</a>
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
						<span className="brand__mark" aria-hidden="true" />
						<span className="brand__name">StageFlow</span>
					</Link>
					<p className="muted site-footer__tagline">
						Self-hostable frontend quality platform.
					</p>
				</div>
				<div className="site-footer__cols">
					<nav aria-label="Product" className="site-footer__col">
						<span className="label">Product</span>
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
