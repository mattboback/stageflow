import type { Config } from '@react-router/dev/config';

export default {
	appDirectory: 'app',
	// SPA mode: no runtime server. Public, indexable pages are prerendered to
	// static HTML; dynamic /scan/* routes hydrate client-side from the SPA shell.
	ssr: false,
	prerender: ['/', '/projects', '/playground']
	// The v8_* future flags this config used to opt into are all standard in
	// React Router 8: middleware, the Vite Environment API, pass-through
	// requests, and trailing-slash-aware data requests are now always on, and
	// splitRouteModules moved to a top-level field that already defaults to the
	// `true` we were asking for.
} satisfies Config;
