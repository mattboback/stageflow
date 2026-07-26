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
import { THEME_INIT_SCRIPT } from './lib/theme';

import './styles/instrument.css';
import './styles/fault.css';

export const links: LinksFunction = () => [
	{ rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' },
	{ rel: 'manifest', href: '/site.webmanifest' }
];

export function Layout({ children }: { children: React.ReactNode }) {
	return (
		/*
		 * suppressHydrationWarning is required, not tidiness. THEME_INIT_SCRIPT
		 * sets data-theme on this element before React hydrates, and React 19
		 * reports the extra attribute through console.error — which the
		 * Playwright fixture fails every spec on.
		 *
		 * This lives in Layout rather than a route because Layout also produces
		 * __spa-fallback.html, which nginx serves for /scan/* and /demo. One
		 * place covers every entry document.
		 */
		<html lang="en" suppressHydrationWarning>
			<head>
				<meta charSet="utf-8" />
				<meta name="viewport" content="width=device-width, initial-scale=1" />
				<Meta />
				<Links />
				<script dangerouslySetInnerHTML={{ __html: THEME_INIT_SCRIPT }} />
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
