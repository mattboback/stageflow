import { Link } from 'react-router';

type Current = 'home' | 'playground' | undefined;

const GITHUB_URL = 'https://github.com/mattboback/stageflow';
const DOCS_URL = `${GITHUB_URL}/tree/main/docs`;

export function SiteHeader({ current }: { current?: Current }) {
	return (
		<>
			<header className="site-header">
				<div className="wrap site-header__bar">
					<Link className="brand" to="/" aria-label="StageFlow home">
						<span className="brand__mark" aria-hidden="true" />
						<span className="brand__name">StageFlow</span>
					</Link>
					<nav className="nav" aria-label="Primary">
						<Link
							className="navlink"
							to="/playground"
							aria-current={current === 'playground' ? 'page' : undefined}
						>
							Configure scan
						</Link>
						<Link className="navlink" to="/#scanners">
							Scanners
						</Link>
						<a className="navlink" href={DOCS_URL}>
							Docs
						</a>
						<a className="btn btn--ghost btn--sm" href={GITHUB_URL}>
							GitHub ↗
						</a>
					</nav>
				</div>
			</header>
		</>
	);
}
