import type { Config } from '@react-router/dev/config';

export default {
	appDirectory: 'app',
	// SPA mode: no runtime server. Public, indexable pages are prerendered to
	// static HTML; dynamic /scan/* routes hydrate client-side from the SPA shell.
	ssr: false,
	prerender: ['/', '/projects', '/playground'],
	future: {
		v8_middleware: true,
		v8_splitRouteModules: true,
		v8_viteEnvironmentApi: true,
		v8_passThroughRequests: true,
		v8_trailingSlashAwareDataRequests: true
	}
} satisfies Config;
