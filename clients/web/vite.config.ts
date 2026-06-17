import { reactRouter } from '@react-router/dev/vite';
import { defineConfig } from 'vite';

// https://reactrouter.com/start/framework/installation
export default defineConfig({
	plugins: [reactRouter()],
	server: {
		port: 3020,
		strictPort: true
	}
});
