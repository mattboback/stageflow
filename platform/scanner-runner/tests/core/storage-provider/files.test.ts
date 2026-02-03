import { beforeEach, describe, expect, it, vi } from "vitest";

import { getFileSize, waitForFileReady } from "../../../src/core/storage-provider/files";

// Mock node:fs/promises
vi.mock("node:fs/promises", () => ({
  stat: vi.fn(),
}));

import { stat } from "node:fs/promises";

const mockLogger = {
  debug: vi.fn(),
  info: vi.fn(),
  warn: vi.fn(),
  error: vi.fn(),
};

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
});

describe("waitForFileReady", () => {
  it("returns true when file exists and has content", async () => {
    vi.mocked(stat).mockResolvedValue({ size: 1000 } as any);

    const result = await waitForFileReady("/path/to/file.txt", mockLogger as any);

    expect(result).toBe(true);
    expect(stat).toHaveBeenCalledWith("/path/to/file.txt");
  });

  it("returns false after max retries when file is empty", async () => {
    vi.mocked(stat).mockResolvedValue({ size: 0 } as any);

    const promise = waitForFileReady("/path/to/empty.txt", mockLogger as any, 3);

    // Fast-forward through all retries
    await vi.runAllTimersAsync();

    const result = await promise;

    expect(result).toBe(false);
    expect(stat).toHaveBeenCalledTimes(3);
    expect(mockLogger.debug).toHaveBeenCalledWith(
      "File is empty, retrying",
      expect.objectContaining({ filePath: "/path/to/empty.txt" }),
    );
  });

  it("returns false after max retries when file does not exist", async () => {
    vi.mocked(stat).mockRejectedValue(new Error("ENOENT: no such file"));

    const promise = waitForFileReady("/path/to/missing.txt", mockLogger as any, 3);

    await vi.runAllTimersAsync();

    const result = await promise;

    expect(result).toBe(false);
    expect(stat).toHaveBeenCalledTimes(3);
    expect(mockLogger.debug).toHaveBeenCalledWith(
      "File not ready",
      expect.objectContaining({
        filePath: "/path/to/missing.txt",
        error: "ENOENT: no such file",
      }),
    );
  });

  it("retries with increasing delay", async () => {
    vi.mocked(stat)
      .mockRejectedValueOnce(new Error("not ready"))
      .mockRejectedValueOnce(new Error("not ready"))
      .mockResolvedValue({ size: 500 } as any);

    const promise = waitForFileReady("/path/to/file.txt", mockLogger as any);

    // First attempt fails immediately
    await vi.advanceTimersByTimeAsync(0);
    expect(stat).toHaveBeenCalledTimes(1);

    // Wait 100ms for second attempt
    await vi.advanceTimersByTimeAsync(100);
    expect(stat).toHaveBeenCalledTimes(2);

    // Wait 200ms for third attempt
    await vi.advanceTimersByTimeAsync(200);
    expect(stat).toHaveBeenCalledTimes(3);

    const result = await promise;
    expect(result).toBe(true);
  });

  it("uses default maxRetries of 5", async () => {
    vi.mocked(stat).mockRejectedValue(new Error("always fails"));

    const promise = waitForFileReady("/path/to/file.txt", mockLogger as any);

    await vi.runAllTimersAsync();

    await promise;

    expect(stat).toHaveBeenCalledTimes(5);
  });

  it("returns true immediately when file is ready on first try", async () => {
    vi.mocked(stat).mockResolvedValue({ size: 100 } as any);

    const result = await waitForFileReady("/path/to/ready.txt", mockLogger as any);

    expect(result).toBe(true);
    expect(stat).toHaveBeenCalledTimes(1);
  });

  it("handles non-Error exceptions", async () => {
    vi.mocked(stat).mockRejectedValue("string error");

    const promise = waitForFileReady("/path/to/file.txt", mockLogger as any, 1);

    await vi.runAllTimersAsync();

    await promise;

    expect(mockLogger.debug).toHaveBeenCalledWith(
      "File not ready",
      expect.objectContaining({ error: "string error" }),
    );
  });
});

describe("getFileSize", () => {
  it("returns file size when file exists", async () => {
    vi.mocked(stat).mockResolvedValue({ size: 2048 } as any);

    const size = await getFileSize("/path/to/file.txt");

    expect(size).toBe(2048);
    expect(stat).toHaveBeenCalledWith("/path/to/file.txt");
  });

  it("returns 0 when file does not exist", async () => {
    vi.mocked(stat).mockRejectedValue(new Error("ENOENT"));

    const size = await getFileSize("/path/to/missing.txt");

    expect(size).toBe(0);
  });

  it("returns 0 on any error", async () => {
    vi.mocked(stat).mockRejectedValue(new Error("Permission denied"));

    const size = await getFileSize("/path/to/protected.txt");

    expect(size).toBe(0);
  });

  it("returns size for empty file", async () => {
    vi.mocked(stat).mockResolvedValue({ size: 0 } as any);

    const size = await getFileSize("/path/to/empty.txt");

    expect(size).toBe(0);
  });

  it("handles large file sizes", async () => {
    const largeSize = 10 * 1024 * 1024 * 1024; // 10GB
    vi.mocked(stat).mockResolvedValue({ size: largeSize } as any);

    const size = await getFileSize("/path/to/large.bin");

    expect(size).toBe(largeSize);
  });
});
