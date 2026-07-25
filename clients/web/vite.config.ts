import { reactRouter } from '@react-router/dev/vite';
import { defineConfig } from 'vite';

// https://reactrouter.com/start/framework/installation
export default defineConfig({
	plugins: [reactRouter()],
	server: {
		port: 3020,
		strictPort: true
	},
	preview: {
		// Prerendering in React Router 8 starts a Vite preview server and then issues
		// node:http requests to `previewServer.resolvedUrls.local[0]`. Left to Vite's
		// default the server binds via `localhost`, so the bind address and the dial
		// address are resolved separately and can disagree: the container image build
		// failed with `Prerender: Request failed for /: connect ECONNREFUSED
		// 127.0.0.1:33023` while the identical build succeeded locally. Binding an
		// explicit IPv4 loopback makes both ends the same literal address.
		host: '127.0.0.1'
	}
});
