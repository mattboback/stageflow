import { createLogger } from './utils/logger';
import { runWorkerMode } from './worker';

const logger = createLogger('main');

async function main() {
	// The Scanner Runner (scanner-runner) now only runs in worker mode.
	// Job submission is handled by the orchestrator API
	await runWorkerMode();
}

main().catch((err: unknown) => {
	const error = err instanceof Error ? err : new Error(String(err));
	logger.error('Scanner Runner (scanner-runner) encountered an error:', { error: error.message });
	process.exit(1);
});
