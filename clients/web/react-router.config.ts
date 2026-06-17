import type { Config } from '@react-router/dev/config';

export default {
	appDirectory: 'app',
	// SPA mode: no runtime server. The marketing landing is prerendered to static
	// HTML at build time (real <h1> + JSON-LD for SEO); dynamic /scan/* routes
	// hydrate client-side from the prerendered shell.
	ssr: false,
	prerender: ['/']
} satisfies Config;
