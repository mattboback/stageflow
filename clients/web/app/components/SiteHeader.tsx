import { Link } from 'react-router';
import { DOCS_URL, GITHUB_URL, SITE_NAME } from '../lib/site-metadata';
import { BrandMark } from './BrandMark';
import { MobileNav } from './MobileNav';
import { ThemeToggle } from './ThemeToggle';

type Current = 'home' | 'projects' | 'playground' | undefined;

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
					<Link className="brand" to="/" aria-label={`${SITE_NAME} home`}>
						<BrandMark />
						<span className="brand__name">{SITE_NAME}</span>
					</Link>
					{app.section && <span className="site-header__section">{app.section}</span>}
					<nav className="nav" aria-label="Workflow">
						<Link className="navlink navlink--back" to={app.backTo}>
							<span aria-hidden="true">←</span> {app.backLabel}
						</Link>
						<ThemeToggle />
					</nav>
				</div>
			</header>
		);
	}

	const links = (
		<>
			<Link
				className="navlink"
				to="/projects"
				aria-current={current === 'projects' ? 'page' : undefined}
			>
				Projects
			</Link>
			<Link
				className="navlink"
				to="/playground"
				aria-current={current === 'playground' ? 'page' : undefined}
			>
				Configure scan
			</Link>
			<Link className="navlink" to="/demo">
				Demo report
			</Link>
			<Link className="navlink" to="/#scanners">
				Scanners
			</Link>
			<a className="navlink" href={DOCS_URL}>
				Docs
			</a>
		</>
	);

	return (
		<header className="site-header">
			<div className="wrap site-header__bar">
				<Link className="brand" to="/" aria-label={`${SITE_NAME} home`}>
					<BrandMark />
					<span className="brand__name">{SITE_NAME}</span>
				</Link>
				<nav className="nav" aria-label="Primary">
					{/* One definition, two placements. Only ever one of them is
					    interactive: the inline row is display:none below 640px, which
					    takes it out of the tab order, and the panel's children do not
					    exist at all until it is opened. */}
					<div className="navlinks">{links}</div>
					<ThemeToggle />
					<a className="btn btn--ghost btn--sm navgh" href={GITHUB_URL}>
						GitHub ↗
					</a>
					<MobileNav>
						{links}
						<a className="navlink" href={GITHUB_URL}>
							GitHub ↗
						</a>
					</MobileNav>
				</nav>
			</div>
		</header>
	);
}
