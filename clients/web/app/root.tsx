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
	{ rel: 'preconnect', href: 'https://fonts.googleapis.com' },
	{ rel: 'preconnect', href: 'https://fonts.gstatic.com', crossOrigin: 'anonymous' },
	{
		rel: 'stylesheet',
		href: 'https://fonts.googleapis.com/css2?family=Gabarito:wght@500;600;700;800&family=Source+Sans+3:ital,wght@0,400;0,500;0,600;0,700;1,400&family=JetBrains+Mono:wght@400;500&display=swap'
	}
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
		<main style={{ padding: '4rem 1.5rem', maxWidth: '32rem', margin: '0 auto' }}>
			<h1>{title}</h1>
			<p>
				{status === 404
					? 'The page you requested could not be found.'
					: 'StageFlow could not render this page. Please try again.'}
			</p>
		</main>
	);
}
