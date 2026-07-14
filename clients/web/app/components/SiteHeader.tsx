import { Link } from 'react-router';

type Current = 'home' | 'playground' | undefined;

const GITHUB_URL = 'https://github.com/mattboback/stageflow';
const DOCS_URL = `${GITHUB_URL}/tree/main/docs`;

/* Application chrome for scan/report workflows: brand, current section, and a
   back action. Marketing nav and GitHub stay on marketing pages only. */
interface AppBar {
	backTo: string;
	backLabel: string;
	section?: string;
}

export function SiteHeader({ current, app }: { current?: Current; app?: AppBar }) {
	if (app) {
		return (
			<header className="site-header site-header--app">
				<div className="wrap wrap--app site-header__bar">
					<Link className="brand" to="/" aria-label="StageFlow home">
						<span className="brand__mark" aria-hidden="true" />
						<span className="brand__name">StageFlow</span>
					</Link>
					{app.section && <span className="site-header__section">{app.section}</span>}
					<nav className="nav" aria-label="Workflow">
						<Link className="navlink navlink--back" to={app.backTo}>
							<span aria-hidden="true">←</span> {app.backLabel}
						</Link>
					</nav>
				</div>
			</header>
		);
	}

	return (
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
	);
}
