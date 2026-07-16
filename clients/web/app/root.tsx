import {
	Links,
	Meta,
	Outlet,
	Scripts,
	ScrollRestoration,
	isRouteErrorResponse,
	type LinksFunction
} from 'react-router';

import './styles/instrument.css';

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

export function ErrorBoundary({ error }: { error: unknown }) {
	const status = isRouteErrorResponse(error) ? error.status : 500;
	const title = status === 404 ? 'Page not found' : 'Something went wrong';
	return (
		<main id="main" style={{ padding: '4rem 1.5rem', maxWidth: '32rem', margin: '0 auto' }}>
			<h1>{title}</h1>
			<p>
				{status === 404
					? 'The page you requested could not be found.'
					: 'StageFlow could not render this page. Please try again.'}
			</p>
		</main>
	);
}
