import { createReadStream, existsSync, statSync } from 'node:fs';
import { createServer } from 'node:http';
import { extname, join, normalize, resolve } from 'node:path';

const root = resolve(process.cwd(), 'build/client');
const fallback = join(root, '__spa-fallback.html');
const port = Number(process.env.PORT || 4173);
const contentTypes = {
	'.css': 'text/css; charset=utf-8',
	'.html': 'text/html; charset=utf-8',
	'.ico': 'image/x-icon',
	'.js': 'text/javascript; charset=utf-8',
	'.json': 'application/json; charset=utf-8',
	'.png': 'image/png',
	'.svg': 'image/svg+xml',
	'.webp': 'image/webp',
	'.txt': 'text/plain; charset=utf-8',
	'.webmanifest': 'application/manifest+json',
	/* Not optional. This server sends X-Content-Type-Options: nosniff, so a
	   font served as application/octet-stream is refused outright and the page
	   silently falls back to system-ui. */
	'.woff2': 'font/woff2'
};

// Mirrors nginx.conf: real files and prerendered pages, the SPA shell for
// scan and project workspace routes, and a 404 carrying the shell for anything else.
const spaRoute = /^\/(?:scan\/[^/]+(?:\/report)?|projects\/[^/]+)\/?$/;

function candidateFor(pathname) {
	if (spaRoute.test(pathname)) return { file: fallback, status: 200 };
	const decoded = decodeURIComponent(pathname);
	const relative = normalize(decoded).replace(/^[/\\]+/, '');
	const candidate = resolve(root, relative);
	if (candidate !== root && !candidate.startsWith(`${root}/`)) return null;
	if (existsSync(candidate) && statSync(candidate).isFile())
		return { file: candidate, status: 200 };
	const index = join(candidate, 'index.html');
	if (existsSync(index) && statSync(index).isFile()) return { file: index, status: 200 };
	return { file: fallback, status: 404 };
}

createServer((request, response) => {
	try {
		const url = new URL(request.url || '/', `http://${request.headers.host || 'localhost'}`);
		const match = candidateFor(url.pathname);
		if (!match || !existsSync(match.file)) {
			response.writeHead(404).end('Not found');
			return;
		}
		const { file, status } = match;
		const type = contentTypes[extname(file)] || 'application/octet-stream';
		response.writeHead(status, {
			'Content-Type': type,
			'Cache-Control': type.startsWith('text/html') ? 'no-cache' : 'public, max-age=0',
			'X-Content-Type-Options': 'nosniff',
			'X-Frame-Options': 'DENY'
		});
		if (request.method === 'HEAD') response.end();
		else createReadStream(file).pipe(response);
	} catch {
		response.writeHead(400).end('Bad request');
	}
}).listen(port, '127.0.0.1', () => {
	process.stdout.write(`StageFlow build server listening on http://127.0.0.1:${port}\n`);
});
