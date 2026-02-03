import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ScannerLogger, StorageConfig } from "../../src/core/types";

import {
  MinioStorageProvider,
  parseMinioEndpoint,
} from "../../src/core/storage-provider";

const mocks = vi.hoisted(() => {
  const client = {
    bucketExists: vi.fn(),
    makeBucket: vi.fn(),
    fPutObject: vi.fn(),
    putObject: vi.fn(),
    fGetObject: vi.fn(),
    statObject: vi.fn(),
  };
  return {
    client,
    Client: vi.fn(function ClientMock() {
      return client;
    }),
  };
});

vi.mock("minio", () => ({
  Client: mocks.Client,
}));

const logger: ScannerLogger = {
  info: vi.fn(),
  warn: vi.fn(),
  error: vi.fn(),
  debug: vi.fn(),
};

const baseConfig: StorageConfig = {
  endpoint: "localhost:9000",
  accessKey: "minio",
  secretKey: "minio123",
  useSSL: false,
  bucket: "scanner-artifacts",
};

describe("parseMinioEndpoint", () => {
  it("parses endpoints with explicit ports", () => {
    expect(parseMinioEndpoint("minio.local:9443", true)).toEqual({
      endPoint: "minio.local",
      port: 9443,
    });
  });

  it("defaults port based on SSL when not provided", () => {
    expect(parseMinioEndpoint("minio.local", false)).toEqual({
      endPoint: "minio.local",
      port: 9000,
    });
    expect(parseMinioEndpoint("minio.local", true)).toEqual({
      endPoint: "minio.local",
      port: 443,
    });
  });
});

describe("MinioStorageProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("retries bucketExists on retryable errors before creating the bucket", async () => {
    const retryableError = { code: "ECONNRESET", message: "reset" };

    mocks.client.bucketExists
      .mockRejectedValueOnce(retryableError)
      .mockRejectedValueOnce(retryableError)
      .mockResolvedValue(false);
    mocks.client.makeBucket.mockResolvedValue(undefined);

    const provider = new MinioStorageProvider(baseConfig, logger);

    vi.useFakeTimers();
    try {
      const ensurePromise = provider.ensureBucket("scanner-artifacts");
      await vi.runAllTimersAsync();
      await ensurePromise;
    } finally {
      vi.useRealTimers();
    }

    expect(mocks.client.bucketExists).toHaveBeenCalledTimes(3);
    expect(mocks.client.makeBucket).toHaveBeenCalledWith("scanner-artifacts", "");
  });

  it("bails out on non-retryable errors", async () => {
    mocks.client.bucketExists.mockRejectedValueOnce({
      code: "EINVAL",
      message: "bad request",
    });

    const provider = new MinioStorageProvider(baseConfig, logger);

    await expect(provider.ensureBucket("scanner-artifacts")).rejects.toThrow(
      "MinIO bucketExists failed",
    );
    expect(mocks.client.bucketExists).toHaveBeenCalledTimes(1);
    expect(mocks.client.makeBucket).not.toHaveBeenCalled();
  });
});
