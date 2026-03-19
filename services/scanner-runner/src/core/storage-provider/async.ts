import type { ScannerLogger } from "../types";

import {
	describeMinioError,
	isRetryableMinioError,
	wrapMinioError,
} from "./minio-errors";

const DEFAULT_RETRY_ATTEMPTS = 3;
const DEFAULT_RETRY_DELAY_MS = 500;

export async function withTimeout<T>(
	promise: Promise<T>,
	ms: number,
	errorMessage: string,
): Promise<T> {
	let timeoutHandle: NodeJS.Timeout | undefined;
	const timeoutPromise = new Promise<never>((_, reject) => {
		timeoutHandle = setTimeout(() => {
			reject(new Error(errorMessage));
		}, ms);
	});

	try {
		return await Promise.race([promise, timeoutPromise]);
	} finally {
		if (timeoutHandle) {
			clearTimeout(timeoutHandle);
		}
	}
}

export async function withRetry<T>({
	action,
	fn,
	logger,
	connection,
	context = {},
	attempts = DEFAULT_RETRY_ATTEMPTS,
	baseDelayMs = DEFAULT_RETRY_DELAY_MS,
}: {
	action: string;
	fn: () => Promise<T>;
	logger: ScannerLogger;
	connection: { endpoint: string; port: number; useSSL: boolean };
	context?: Record<string, unknown>;
	attempts?: number;
	baseDelayMs?: number;
}): Promise<T> {
	let lastError: unknown;
	for (let attempt = 1; attempt <= attempts; attempt += 1) {
		try {
			return await fn();
		} catch (err) {
			lastError = err;
			const retryable = isRetryableMinioError(err);
			if (!retryable || attempt >= attempts) {
				break;
			}

			const delayMs = baseDelayMs * 2 ** (attempt - 1);
			logger.warn("MinIO request failed, retrying", {
				action,
				attempt,
				maxAttempts: attempts,
				delayMs,
				...context,
				endpoint: connection.endpoint,
				port: connection.port,
				useSSL: connection.useSSL,
				error: describeMinioError(err),
			});
			await new Promise((resolve) => setTimeout(resolve, delayMs));
		}
	}

	throw wrapMinioError(connection, action, lastError, context);
}
