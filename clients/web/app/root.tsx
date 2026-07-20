import {
	Links,
	Meta,
	Outlet,
	Scripts,
	ScrollRestoration,
	isRouteErrorResponse,
	type LinksFunction
} from 'react-router';

import { SiteHeader } from './components/SiteHeader';
import { SiteFooter } from './components/SiteFooter';
import { RouteFault } from './components/RouteFault';
import { Pill } from './components/Pill';

import './styles/instrument.css';
import './styles/fault.css';

export const links: LinksFunction = () => [
	{ rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' },
	{ rel: 'manifest', href: '/site.webmanifest' }
];

export function Layout({ children }: { children: React.ReactNode }) {
	return (
		<html lang="en">
			<head>
				<meta charSet="utf-8" />
				<meta name="viewport" content="width=device-width, initial-scale=1" />
				<Meta />
				<Links />
			</head>
			<body>
				<a className="skip-link" href="#main">
					Skip to main content
				</a>
				{children}
				<ScrollRestoration />
				<Scripts />
			</body>
		</html>
	);
}

export default function Root() {
	return <Outlet />;
}

export function HydrateFallback() {
	return (
		<>
			<SiteHeader />
			<main id="main" className="hydrate-fallback" role="status" aria-live="polite">
				<div className="hydrate-fallback__panel">
					<Pill variant="queued">Loading</Pill>
					<h1>Loading StageFlow…</h1>
					<p>Fetching this page.</p>
				</div>
			</main>
			<SiteFooter slim />
		</>
	);
}

export function ErrorBoundary({ error }: { error: unknown }) {
	const status = isRouteErrorResponse(error) ? error.status : 500;
	const notFound = status === 404;
	return (
		<>
			<SiteHeader />
			<RouteFault
				status={status}
				title={notFound ? 'Page not found.' : 'Something went wrong.'}
				detail={
					notFound
						? "The page you requested doesn't exist, has expired, or has moved."
						: 'StageFlow could not render this page. Try again, or head back home.'
				}
				traceKey="renderer"
				traceLine={notFound ? 'route not matched' : 'route failed to render'}
				traceHint={notFound ? 'check the URL or run a new scan' : 'reload the page or return home'}
			/>
			<SiteFooter slim />
		</>
	);
}
