import { access } from 'node:fs/promises';

async function main(): Promise<void> {
	const dataDir = process.env.SCANNER_DATA_DIR ?? '/data';
	await access(dataDir);
}

main().catch((error: unknown) => {
	const message = error instanceof Error ? error.message : String(error);
	console.error(`scanner-runner healthcheck failed: ${message}`);
	process.exit(1);
});
