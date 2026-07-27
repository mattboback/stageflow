import {
	Link,
	isRouteErrorResponse,
	useLocation,
	useRouteError,
	type MetaFunction
} from 'react-router';

import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { RouteFault } from '../components/RouteFault';
import { DemoArtifactsPanel } from '../components/report/DemoArtifactsPanel';
import { ReportView } from '../components/report/ReportView';
import {
	DEMO_JOB_ID,
	DEMO_REPORT_URL,
	DEMO_SCREENSHOTS_URL,
	useDemoReport
} from '../lib/hooks/useDemoReport';
import reportStyles from './scan-report.css?url';
import severityStyles from '../styles/report.css?url';
import { pageTitle, SITE_NAME } from '../lib/site-metadata';

export const links = () => [
	{ rel: 'stylesheet', href: severityStyles },
	{ rel: 'stylesheet', href: reportStyles },
	/*
	 * Both fetches start while the shell is still rendering rather than after
	 * the effect runs. links() is used instead of a clientLoader because the
	 * eslint config's allowExportNames makes a clientLoader export a hard error
	 * in a route module -- and a preload is the cheaper mechanism anyway.
	 */
	{ rel: 'preload', href: DEMO_REPORT_URL, as: 'fetch', type: 'application/json' },
	{ rel: 'preload', href: DEMO_SCREENSHOTS_URL, as: 'fetch', type: 'application/json' }
];

export const meta: MetaFunction = () => [
	{ title: pageTitle('Live report demo') },
	{
		name: 'description',
		content: `A real ${SITE_NAME} report you can explore without running a scan: three pages, seven scanners, one merged and severity-ranked result.`
	}
];

/*
 * The product, at the front door.
 *
 * Everything here is the same code path a real scan takes -- ReportView, the
 * review queue, the screenshot overlays, the contrast eyedropper -- fed from a
 * committed fixture instead of an API. See devtools/scripts/build-demo-fixture.sh
 * for where the fixture comes from: it is a genuine scan of this site, not a
 * hand-written mock.
 *
 * Deliberately absent from `prerender` in react-router.config.ts. ReportHeader
 * and review-verdict both call toLocaleString(undefined, ...): prerendering
 * runs in UTC under Node and the browser renders in the visitor's locale, so
 * the two disagree, React reports a hydration mismatch through console.error,
 * and the Playwright fixture fails every spec. It is served from
 * __spa-fallback.html like /scan/:id/report.
 */
export default function Demo() {
	const { report, screenshots, status, error } = useDemoReport();

	return (
		<>
			{/* Marketing header, not the workflow back-bar: a visitor lands here
			    from the front door and should keep the primary navigation. */}
			<SiteHeader current="demo" />
			<ReportView
				jobId={DEMO_JOB_ID}
				report={report}
				screenshots={screenshots}
				status={status}
				error={error}
				banner={null}
				artifactsPanel={<DemoArtifactsPanel />}
			/>
			<SiteFooter slim />
		</>
	);
}

export function ErrorBoundary() {
	const routeError = useRouteError();
	const { pathname } = useLocation();
	const status = isRouteErrorResponse(routeError) ? routeError.status : 500;

	return (
		<>
			<SiteHeader current="demo" />
			<RouteFault
				status={status}
				title="The demo report can't be shown."
				detail="The demo renders a committed report with no API behind it, so this is a deployment problem rather than a scan that failed."
				traceLine={`demo view ${pathname} failed to load`}
				traceHint="reload the page, or run a scan of your own"
				actions={
					<>
						<Link className="btn btn--primary" to="/playground">
							Run a real scan{' '}
							<span className="ar" aria-hidden="true">
								→
							</span>
						</Link>
						<Link className="btn btn--ghost" to="/">
							Back home
						</Link>
					</>
				}
			/>
			<SiteFooter slim />
		</>
	);
}
