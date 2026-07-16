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
	'.txt': 'text/plain; charset=utf-8',
	'.webmanifest': 'application/manifest+json'
};

function candidateFor(pathname) {
	const decoded = decodeURIComponent(pathname);
	const relative = normalize(decoded).replace(/^[/\\]+/, '');
	const candidate = resolve(root, relative);
	if (candidate !== root && !candidate.startsWith(`${root}/`)) return null;
	if (existsSync(candidate) && statSync(candidate).isFile()) return candidate;
	const index = join(candidate, 'index.html');
	if (existsSync(index) && statSync(index).isFile()) return index;
	return fallback;
}

createServer((request, response) => {
	try {
		const url = new URL(request.url || '/', `http://${request.headers.host || 'localhost'}`);
		const file = candidateFor(url.pathname);
		if (!file || !existsSync(file)) {
			response.writeHead(404).end('Not found');
			return;
		}
		const type = contentTypes[extname(file)] || 'application/octet-stream';
		response.writeHead(200, {
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
