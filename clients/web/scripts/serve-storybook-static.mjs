import { createReadStream } from "node:fs";
import { stat } from "node:fs/promises";
import { createServer } from "node:http";
import { extname, join, normalize } from "node:path";

const host = "127.0.0.1";
const port = Number.parseInt(process.env.PORT ?? "6006", 10);
const rootDir = join(process.cwd(), "storybook-static");

const mimeTypes = {
	".css": "text/css; charset=utf-8",
	".html": "text/html; charset=utf-8",
	".js": "text/javascript; charset=utf-8",
	".json": "application/json; charset=utf-8",
	".map": "application/json; charset=utf-8",
	".png": "image/png",
	".svg": "image/svg+xml",
	".txt": "text/plain; charset=utf-8",
	".woff2": "font/woff2",
};

function resolvePathname(urlPath) {
	const decodedPath = decodeURIComponent(urlPath);
	const normalizedPath = normalize(decodedPath)
		.replace(/^(\.\.(\/|\\|$))+/, "")
		.replace(/^([/\\])+/, "");
	const withIndex = normalizedPath.endsWith("/")
		? join(normalizedPath, "index.html")
		: normalizedPath;

	return join(rootDir, withIndex);
}

async function selectFilePath(urlPath) {
	const requestedPath = resolvePathname(urlPath);

	try {
		const requestedStat = await stat(requestedPath);
		if (requestedStat.isDirectory()) {
			return join(requestedPath, "index.html");
		}
		return requestedPath;
	} catch {
		if (extname(requestedPath).length > 0) {
			return null;
		}

		// Support deep links in the Storybook manager UI.
		return join(rootDir, "index.html");
	}
}

const server = createServer(async (req, res) => {
	try {
		const requestUrl = new URL(req.url ?? "/", `http://${host}:${port}`);
		const filePath = await selectFilePath(requestUrl.pathname);
		if (filePath === null) {
			res.writeHead(404, { "Content-Type": "text/plain; charset=utf-8" });
			res.end("Not found");
			return;
		}

		const extension = extname(filePath);
		const contentType = mimeTypes[extension] ?? "application/octet-stream";

		res.writeHead(200, { "Content-Type": contentType });
		createReadStream(filePath).pipe(res);
	} catch (error) {
		const message =
			error instanceof Error ? error.message : "Unknown server error";
		res.writeHead(500, { "Content-Type": "text/plain; charset=utf-8" });
		res.end(`Failed to serve Storybook: ${message}`);
	}
});

server.listen(port, host, () => {
	console.log(`Storybook static server running at http://${host}:${port}`);
});
