import { Link, useLocation, type MetaFunction } from 'react-router';
import { SiteHeader } from '../components/SiteHeader';
import { Pill } from '../components/Pill';
import notFoundStyles from './not-found.css?url';

export const links = () => [{ rel: 'stylesheet', href: notFoundStyles }];

export const meta: MetaFunction = () => [
	{ title: '404 · Channel not found — StageFlow' },
	{ name: 'robots', content: 'noindex' }
];

export default function NotFound() {
	const { pathname } = useLocation();

	return (
		<>
			<SiteHeader />

			<main className="fault">
				<div className="panel fault__panel" style={{ boxShadow: 'var(--shadow-md)' }}>
					<div className="fault__top">
						<span className="label">Fault</span>
						<Pill variant="error" style={{ marginLeft: 'auto' }}>
							404
						</Pill>
					</div>
					<div className="fault__code">
						4<span>0</span>4
					</div>
					<div className="fault__body">
						<h1>This channel isn't on the bench.</h1>
						<p>
							The page or scan you requested doesn't exist, expired, or moved. Scan jobs are kept
							for a limited window after they complete.
						</p>
						<div className="fault__trace" role="status">
							<div>
								<span className="k">resolver</span> route {pathname} not matched
							</div>
							<div>
								<span className="warn">→</span> no record · check the job id or run a new scan
							</div>
						</div>
						<div className="fault__actions">
							<Link className="btn btn--primary" to="/">
								Back to home{' '}
								<span className="ar" aria-hidden="true">
									→
								</span>
							</Link>
							<Link className="btn btn--ghost" to="/playground">
								Run a new scan
							</Link>
						</div>
					</div>
				</div>
			</main>
		</>
	);
}
