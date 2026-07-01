import type { Config } from '@react-router/dev/config';

export default {
	appDirectory: 'app',
	// SPA mode: no runtime server. Public, indexable pages are prerendered to
	// static HTML; dynamic /scan/* routes hydrate client-side from the SPA shell.
	ssr: false,
	prerender: ['/', '/playground']
} satisfies Config;
