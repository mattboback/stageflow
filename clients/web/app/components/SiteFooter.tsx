import { Link } from 'react-router';

const GITHUB_URL = 'https://github.com/mattboback/stageflow';
const DOCS_URL = `${GITHUB_URL}/tree/main/docs`;
const CLI_REFERENCE_URL = `${GITHUB_URL}/tree/main/docs/reference/cli/stageflow`;
const SELF_HOSTING_URL = `${GITHUB_URL}/blob/main/docs/operations/deployment.md`;

export function SiteFooter() {
	return (
		<footer className="site-footer">
			<div className="wrap site-footer__inner">
				<div className="site-footer__brand">
					<Link className="brand" to="/">
						<span className="brand__mark" aria-hidden="true" />
						<span className="brand__name">StageFlow</span>
					</Link>
					<p className="muted site-footer__tagline">
						Self-hostable frontend quality platform. MIT licensed.
					</p>
				</div>
				<nav aria-label="Product" className="site-footer__col">
					<span className="label">Product</span>
					<Link to="/playground">Playground</Link>
					<Link to="/#scanners">Scanners</Link>
				</nav>
				<nav aria-label="Resources" className="site-footer__col">
					<span className="label">Resources</span>
					<a href={DOCS_URL}>Documentation</a>
					<a href={CLI_REFERENCE_URL}>CLI reference</a>
					<a href={SELF_HOSTING_URL}>Self-hosting</a>
				</nav>
				<nav aria-label="Project" className="site-footer__col">
					<span className="label">Project</span>
					<a href={GITHUB_URL}>GitHub</a>
					<a href={`${GITHUB_URL}/blob/main/CHANGELOG.md`}>Changelog</a>
					<a href={`${GITHUB_URL}/security/policy`}>Security</a>
				</nav>
			</div>
		</footer>
	);
}
