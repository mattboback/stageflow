const RETRYABLE_ERROR_CODES = new Set([
  "ECONNREFUSED",
  "ECONNRESET",
  "ENOTFOUND",
  "EAI_AGAIN",
  "ETIMEDOUT",
  "ENETUNREACH",
  "EHOSTUNREACH",
]);

export function isRetryableMinioError(err: unknown): boolean {
  if (!err || typeof err !== "object") {
    return false;
  }

  const code = (err as { code?: unknown }).code;
  if (typeof code === "string" && RETRYABLE_ERROR_CODES.has(code)) {
    return true;
  }

  const statusCode = (err as { statusCode?: unknown }).statusCode;
  if (typeof statusCode === "number") {
    return statusCode === 429 || statusCode >= 500;
  }

  return false;
}

export function describeMinioError(err: unknown): {
  code?: string;
  statusCode?: number;
  message?: string;
  region?: string;
  requestId?: string;
  hostId?: string;
} {
  if (!err || typeof err !== "object") {
    return { message: String(err) };
  }

  const record = err as {
    code?: unknown;
    message?: unknown;
    statusCode?: unknown;
    region?: unknown;
    amzRequestId?: unknown;
    amzId2?: unknown;
  };

  return {
    code: typeof record.code === "string" ? record.code : undefined,
    statusCode: typeof record.statusCode === "number" ? record.statusCode : undefined,
    message: typeof record.message === "string" ? record.message : (err instanceof Error ? err.message : JSON.stringify(err)),
    region: typeof record.region === "string" ? record.region : undefined,
    requestId: typeof record.amzRequestId === "string" ? record.amzRequestId : undefined,
    hostId: typeof record.amzId2 === "string" ? record.amzId2 : undefined,
  };
}

export function wrapMinioError(
  connection: { endpoint: string; port: number; useSSL: boolean },
  action: string,
  err: unknown,
  context: Record<string, unknown> = {},
): Error {
  const details = describeMinioError(err);
  const message = details.message ?? "unknown error";
  const parts = [
    `endpoint=${connection.endpoint}:${connection.port}`,
    `ssl=${connection.useSSL}`,
    ...Object.entries(context).map(([key, value]) => `${key}=${String(value)}`),
    ...(details.code ? [`code=${details.code}`] : []),
    ...(details.statusCode ? [`status=${details.statusCode}`] : []),
  ];

  return new Error(`MinIO ${action} failed: ${message} (${parts.join(", ")})`);
}
