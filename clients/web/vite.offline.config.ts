import { resolve } from 'node:path';
import { defineConfig } from 'vite';

export default defineConfig({
	define: { 'process.env.NODE_ENV': JSON.stringify('production') },
	build: {
		outDir: 'build/client/assets',
		emptyOutDir: false,
		lib: {
			entry: resolve(import.meta.dirname, 'app/offline/entry.tsx'),
			name: 'StageFlowOffline',
			formats: ['iife'],
			fileName: () => 'offline-viewer.js',
			cssFileName: 'offline-viewer'
		}
	},
	plugins: [
		{
			name: 'offline-fonts',
			transform(source, id) {
				if (!id.endsWith('/instrument.css')) return;
				// System font fallbacks keep the report entirely portable without requests.
				return source.replace(/@font-face\s*\{[^}]*\}/g, '');
			}
		}
	]
});
