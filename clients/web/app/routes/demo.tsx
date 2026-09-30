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
import { BaselineDiffSummary } from '../components/report/BaselineDiffSummary';
import { computeProjectDiff } from '../lib/projects';
import type { UnifiedReport } from '../lib/types/unified-report';
import type { ScreenshotArtifact } from '../lib/types/scan';
import currentData from '../../public/demo/report.json';
import baselineData from '../../public/demo/baseline.json';
import screenshotData from '../../public/demo/screenshots.json';
import { useSearchParams } from 'react-router';
import reportStyles from './scan-report.css?url';
import severityStyles from '../styles/report.css?url';
import { pageTitle, SITE_NAME } from '../lib/site-metadata';

export const links = () => [
	{ rel: 'stylesheet', href: severityStyles },
	{ rel: 'stylesheet', href: reportStyles }
];

export const meta: MetaFunction = () => [
	{ title: pageTitle('Live report demo') },
	{
		name: 'description',
		content: `A real ${SITE_NAME} report you can explore without running a scan: a sample deployment, new and fixed findings, evidence, and concrete fixes.`
	}
];

const report = currentData as unknown as UnifiedReport;
const baseline = baselineData as unknown as UnifiedReport;
const screenshots = screenshotData as ScreenshotArtifact[];
const comparison = computeProjectDiff('demo-baseline', baseline, 'demo', report);

export default function Demo() {
	const [, setSearchParams] = useSearchParams();
	const openIssue = (issueId: string) => {
		setSearchParams(
			(prev) => {
				const next = new URLSearchParams(prev);
				next.set('section', 'issues');
				next.set('issue', issueId);
				return next;
			},
			{ replace: true, preventScrollReset: true }
		);
	};
	return (
		<>
			<SiteHeader current="demo" />
			<ReportView
				jobId="demo"
				report={report}
				screenshots={screenshots}
				status="complete"
				error={null}
				comparison={comparison}
				banner={<BaselineDiffSummary diff={comparison} onOpenIssue={openIssue} />}
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
							Configure a scan{' '}
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
