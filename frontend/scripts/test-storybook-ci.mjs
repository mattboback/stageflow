import { spawn } from 'node:child_process';
import { createServer } from 'node:net';
import process from 'node:process';
import { setTimeout as sleep } from 'node:timers/promises';

const HOST = '127.0.0.1';
const SERVER_POLL_INTERVAL_MS = 200;
const SERVER_READY_TIMEOUT_MS = 30_000;
const SERVER_STOP_TIMEOUT_MS = 5_000;

function getFreePort(host) {
	return new Promise((resolve, reject) => {
		const probe = createServer();

		probe.once('error', reject);
		probe.listen(0, host, () => {
			const address = probe.address();
			if (address === null || typeof address === 'string') {
				probe.close(() => {
					reject(new Error('Failed to determine a free localhost port'));
				});
				return;
			}

			probe.close((closeError) => {
				if (closeError) {
					reject(closeError);
					return;
				}

				resolve(address.port);
			});
		});
	});
}

function spawnProcess(command, args, env = {}) {
	return spawn(command, args, {
		env: { ...process.env, ...env },
		stdio: 'inherit'
	});
}

function spawnBun(args, env = {}) {
	return spawnProcess('bun', args, env);
}

async function waitForServer(url, serverProcess) {
	const deadline = Date.now() + SERVER_READY_TIMEOUT_MS;

	while (Date.now() < deadline) {
		if (serverProcess.exitCode !== null) {
			throw new Error(`Storybook server exited before becoming ready (exit code ${serverProcess.exitCode})`);
		}

		try {
			const response = await fetch(url);
			if (response.ok) {
				return;
			}
		} catch {
			// Server not ready yet.
		}

		await sleep(SERVER_POLL_INTERVAL_MS);
	}

	throw new Error(`Timed out waiting for Storybook server readiness at ${url}`);
}

function waitForExit(processHandle) {
	return new Promise((resolve, reject) => {
		processHandle.once('error', reject);
		processHandle.once('exit', (code, signal) => {
			resolve({ code: code ?? 1, signal });
		});
	});
}

async function runProcess(processHandle) {
	const { code, signal } = await waitForExit(processHandle);
	if (signal) {
		throw new Error(`Process exited from signal ${signal}`);
	}

	return code;
}

async function stopServer(serverProcess) {
	if (serverProcess.exitCode !== null) {
		return;
	}

	serverProcess.kill('SIGTERM');

	const forceKillTimer = setTimeout(() => {
		if (serverProcess.exitCode === null) {
			serverProcess.kill('SIGKILL');
		}
	}, SERVER_STOP_TIMEOUT_MS);

	await waitForExit(serverProcess);
	clearTimeout(forceKillTimer);
}

async function main() {
	const port = await getFreePort(HOST);
	const storybookUrl = `http://${HOST}:${port}`;
	console.log(`Using Storybook test URL: ${storybookUrl}`);

	const serverProcess = spawnProcess('node', ['./scripts/serve-storybook-static.mjs'], {
		PORT: String(port)
	});

	try {
		await waitForServer(storybookUrl, serverProcess);
		const runnerCode = await runProcess(
			spawnBun(['run', 'test-storybook:runner:ci'], { STORYBOOK_URL: storybookUrl })
		);

		if (runnerCode !== 0) {
			throw new Error(`Storybook test runner failed with exit code ${runnerCode}`);
		}
	} finally {
		await stopServer(serverProcess);
	}
}

main().catch((error) => {
	const message = error instanceof Error ? error.message : String(error);
	console.error(`Storybook CI test run failed: ${message}`);
	process.exit(1);
});
