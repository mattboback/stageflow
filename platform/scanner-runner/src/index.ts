import { runWorkerMode } from "./worker";

async function main() {
  // The scan worker (scanner-runner) now only runs in worker mode.
  // Job submission is handled by the orchestrator API
  await runWorkerMode();
}

main().catch((err: unknown) => {
  const error = err instanceof Error ? err : new Error(String(err));
  console.error("scan worker (scanner-runner) encountered an error:", error);
  process.exit(1);
});
