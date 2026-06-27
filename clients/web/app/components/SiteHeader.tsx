import { Link } from 'react-router';

type Current = 'home' | 'playground' | undefined;

const GITHUB_URL = 'https://github.com/mattboback/stageflow';

export function SiteHeader({ current }: { current?: Current }) {
	return (
		<>
			<header className="site-header">
				<div className="wrap site-header__bar">
					<Link className="brand" to="/" aria-label="StageFlow home">
						<span className="brand__mark" aria-hidden="true" />
						<span className="brand__name">STAGEFLOW</span>
					</Link>
					<nav className="nav" aria-label="Primary">
						<Link
							className="navlink"
							to="/playground"
							aria-current={current === 'playground' ? 'page' : undefined}
							style={current === 'playground' ? { color: 'var(--ink-strong)' } : undefined}
						>
							Playground
						</Link>
						<Link className="navlink" to="/#scanners">
							Scanners
						</Link>
						<a className="navlink" href="https://stageflow.org/docs">
							Docs
						</a>
						<a className="btn btn--ghost btn--sm" href={GITHUB_URL} style={{ marginLeft: '.4rem' }}>
							GitHub ↗
						</a>
						<Link className="btn btn--primary btn--sm" to="/playground">
							Run a scan
						</Link>
					</nav>
				</div>
			</header>
			<div className="rule-ticks" aria-hidden="true" />
		</>
	);
}
