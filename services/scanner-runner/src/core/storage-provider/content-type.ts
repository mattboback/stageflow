const CONTENT_TYPE_MAP: Record<string, string> = {
	".json": "application/json",
	".html": "text/html",
	".htm": "text/html",
	".css": "text/css",
	".js": "application/javascript",
	".png": "image/png",
	".jpg": "image/jpeg",
	".jpeg": "image/jpeg",
	".gif": "image/gif",
	".webp": "image/webp",
	".svg": "image/svg+xml",
	".zip": "application/zip",
	".txt": "text/plain",
	".xml": "application/xml",
};

export function guessContentType(filePath: string): string {
	const ext = /\.[^.]+$/.exec(filePath.toLowerCase())?.[0] ?? "";
	return CONTENT_TYPE_MAP[ext] ?? "application/octet-stream";
}
