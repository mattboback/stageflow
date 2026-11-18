import { useLocation, type MetaFunction } from 'react-router';
import { SiteHeader } from '../components/SiteHeader';
import { SiteFooter } from '../components/SiteFooter';
import { RouteFault } from '../components/RouteFault';
import { pageTitle } from '../lib/site-metadata';

export const meta: MetaFunction = () => [
	{ title: pageTitle('404 · Page not found') },
	{ name: 'robots', content: 'noindex' }
];

export default function NotFound() {
	const { pathname } = useLocation();

	return (
		<>
			<SiteHeader />
			<RouteFault
				status={404}
				title="Page not found."
				detail="The page or scan you requested doesn't exist, has expired, or has moved. Scan jobs are kept for a limited window after they complete."
				traceLine={`route ${pathname} not matched`}
				traceHint="no record · check the job id or run a new scan"
			/>
			<SiteFooter slim />
		</>
	);
}
