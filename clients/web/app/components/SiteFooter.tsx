import { Link } from 'react-router';

const GITHUB_URL = 'https://github.com/mattboback/stageflow';

export function SiteFooter() {
	return (
		<footer className="site-footer">
			<div className="wrap site-footer__inner">
				<div style={{ flex: '1 1 14rem' }}>
					<Link className="brand" to="/">
						<span className="brand__mark" aria-hidden="true" />
						<span className="brand__name">STAGEFLOW</span>
					</Link>
					<p className="muted" style={{ fontSize: '.875rem', maxWidth: '32ch', marginTop: '.7rem' }}>
						Self-hostable frontend quality platform. MIT licensed.
					</p>
				</div>
				<nav aria-label="Product" style={{ display: 'grid', gap: '.55rem' }}>
					<span className="label">Product</span>
					<Link to="/playground">Playground</Link>
					<Link to="/#scanners">Scanners</Link>
				</nav>
				<nav aria-label="Resources" style={{ display: 'grid', gap: '.55rem' }}>
					<span className="label">Resources</span>
					<a href="https://stageflow.org/docs">Documentation</a>
					<a href="https://stageflow.org/docs/reference/cli">CLI reference</a>
					<a href="https://stageflow.org/docs/self-hosting">Self-hosting</a>
				</nav>
				<nav aria-label="Project" style={{ display: 'grid', gap: '.55rem' }}>
					<span className="label">Project</span>
					<a href={GITHUB_URL}>GitHub</a>
					<a href={`${GITHUB_URL}/blob/main/CHANGELOG.md`}>Changelog</a>
					<a href={`${GITHUB_URL}/security/policy`}>Security</a>
				</nav>
			</div>
		</footer>
	);
}
