import { type RouteConfig, index, route } from '@react-router/dev/routes';

export default [
	index('routes/home.tsx'),
	route('playground', 'routes/playground.tsx'),
	route('scan/:id', 'routes/scan.tsx'),
	route('scan/:id/report', 'routes/scan-report.tsx'),
	route('*', 'routes/not-found.tsx')
] satisfies RouteConfig;
