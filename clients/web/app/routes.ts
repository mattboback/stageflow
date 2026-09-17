import { type RouteConfig, index, route } from '@react-router/dev/routes';

export default [
	index('routes/home.tsx'),
	route('projects', 'routes/projects.tsx'),
	route('privacy', 'routes/privacy.tsx'),
	route('playground', 'routes/playground.tsx'),
	route('demo', 'routes/demo.tsx'),
	route('scan/:id', 'routes/scan.tsx'),
	route('scan/:id/report', 'routes/scan-report.tsx'),
	route('*', 'routes/not-found.tsx')
] satisfies RouteConfig;
