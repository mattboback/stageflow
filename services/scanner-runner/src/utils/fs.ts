import { access, mkdir, readFile, writeFile } from 'node:fs/promises';

export async function ensureDir(path: string): Promise<void> {
	await mkdir(path, { recursive: true });
}

export async function pathExists(path: string): Promise<boolean> {
	try {
		await access(path);
		return true;
	} catch {
		return false;
	}
}

export async function readJson<T = unknown>(path: string): Promise<T> {
	return JSON.parse(await readFile(path, 'utf8')) as T;
}

export async function writeJson(
	path: string,
	value: unknown,
	options: { spaces?: number } = {}
): Promise<void> {
	await writeFile(path, `${JSON.stringify(value, null, options.spaces)}\n`, 'utf8');
}
