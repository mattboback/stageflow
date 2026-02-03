import fs from "fs-extra";
import { beforeEach, describe, expect, it, type Mock, vi } from "vitest";

import type {
  PageIteratorCallbacks,
  PageScanCallback,
} from "../../src/core/page-iterator";
import type {
  PageScanResult,
  Provenance,
  ScanContext,
  ScannerConfig,
} from "../../src/core/types";

import { ScannerBase, type ScannerMetadata } from "../../src/core/scanner-base";

vi.mock("fs-extra");

const mocks = vi.hoisted(() => {
  const mockBrowserManagerInstance = {
    createContext: vi.fn().mockResolvedValue({
      newPage: vi.fn().mockResolvedValue({
        goto: vi.fn(),
        evaluate: vi.fn(),
        waitForLoadState: vi.fn(),
        waitForTimeout: vi.fn(),
        pdf: vi.fn(),
      }),
      close: vi.fn(),
    }),
    close: vi.fn(),
  };

  const mockPageIteratorInstance = {
    loadProvenance: vi.fn().mockResolvedValue({
      version: "1.0.0",
      job_id: "test-job",
      base_url: "http://localhost:8080",
      pages: [
        { id: "page-1", url: "http://localhost:8080/index.html", path: "/index.html" },
        { id: "page-2", url: "http://localhost:8080/about.html", path: "/about.html" },
      ],
    }),
    iteratePages: vi
      .fn()
      .mockImplementation(
        async (
          provenance: Provenance,
          scannerFn: PageScanCallback,
          hooks: PageIteratorCallbacks = {},
        ) => {
          const results: PageScanResult[] = [];
          const totalPages = provenance.pages.length;
          for (let i = 0; i < totalPages; i++) {
            const page = provenance.pages[i]!;
            const scanContext = {
              pageEntry: page,
            } as unknown as ScanContext;

            if (hooks.onPageStart) {
              await hooks.onPageStart(page, i, totalPages);
            }

            const result = await scannerFn(scanContext);
            results.push(result);

            if (hooks.onPageComplete) {
              await hooks.onPageComplete(result, i, totalPages);
            }
          }
          return results;
        },
      ),
  };

  const mockStorageProviderInstance = {
    ensureBucket: vi.fn().mockResolvedValue(undefined),
    upload: vi.fn().mockResolvedValue(undefined),
  };

  const mockEventPublisherInstance = {
    connect: vi.fn().mockResolvedValue(undefined),
    publishPageCompleted: vi.fn().mockResolvedValue(undefined),
    publishScanCompleted: vi.fn().mockResolvedValue(undefined),
    publishScanFailed: vi.fn().mockResolvedValue(undefined),
    close: vi.fn().mockResolvedValue(undefined),
  };

  return {
    mockBrowserManagerInstance,
    mockPageIteratorInstance,
    mockStorageProviderInstance,
    mockEventPublisherInstance,
  };
});

vi.mock("../../src/core/browser-manager", async () => {
  const actual = await vi.importActual<typeof import("../../src/core/browser-manager")>(
    "../../src/core/browser-manager",
  );
  return {
    ...actual,
    BrowserManager: vi.fn().mockImplementation(function BrowserManagerMock() {
      return mocks.mockBrowserManagerInstance;
    }),
  };
});

vi.mock("../../src/core/page-iterator", async () => {
  const actual = await vi.importActual<typeof import("../../src/core/page-iterator")>(
    "../../src/core/page-iterator",
  );
  return {
    ...actual,
    PageIterator: vi.fn().mockImplementation(function PageIteratorMock() {
      return mocks.mockPageIteratorInstance;
    }),
  };
});

vi.mock("../../src/core/storage-provider", async () => {
  const actual = await vi.importActual<typeof import("../../src/core/storage-provider")>(
    "../../src/core/storage-provider",
  );
  return {
    ...actual,
    MinioStorageProvider: vi.fn().mockImplementation(function MinioMock() {
      return mocks.mockStorageProviderInstance;
    }),
  };
});

vi.mock("../../src/core/event-publisher", async () => {
  const actual = await vi.importActual<typeof import("../../src/core/event-publisher")>(
    "../../src/core/event-publisher",
  );
  return {
    ...actual,
    NatsEventPublisher: vi.fn().mockImplementation(function NatsPublisherMock() {
      return mocks.mockEventPublisherInstance;
    }),
  };
});

class MockScanner extends ScannerBase {
  readonly metadata: ScannerMetadata = {
    name: "mock-scanner",
    version: "1.0.0",
    description: "Mock Scanner",
  };

  scanPage(context: ScanContext): Promise<PageScanResult> {
    return Promise.resolve({
      pageId: context.pageEntry.id,
      url: context.pageEntry.url,
      path: context.pageEntry.path,
      success: true,
      issues: [],
      durationMs: 100,
      startedAt: new Date().toISOString(),
      finishedAt: new Date().toISOString(),
    });
  }
}

describe("ScannerBase", () => {
  let scanner: MockScanner;
  let mockConfig: ScannerConfig;

  beforeEach(() => {
    vi.clearAllMocks();

    scanner = new MockScanner();

    mockConfig = {
      jobId: "test-job",
      provenancePath: "/tmp/provenance.json",
      resultsDir: "/tmp/results",
      scannerName: "mock-scanner",
      concurrency: 1,
      maxRetries: 0,
      browser: {
        headless: true,
        args: [],
        defaultViewport: { width: 1280, height: 720 },
        deviceScaleFactor: 1,
        defaultTimeout: 30000,
        pageLoadTimeout: 30000,
      },
      storage: {
        endpoint: "localhost",
        accessKey: "minio",
        secretKey: "minio123",
        useSSL: false,
        bucket: "test-bucket",
      },
      messaging: {
        url: "nats://localhost:4222",
        subjects: {
          pageCompleted: "page.completed",
          scanCompleted: "scan.completed",
          scanFailed: "scan.failed",
        },
      },
    };

    (fs.ensureDir as unknown as Mock).mockResolvedValue(undefined);
    (fs.writeJSON as unknown as Mock).mockResolvedValue(undefined);
    (fs.writeFile as unknown as Mock).mockResolvedValue(undefined);
    (fs.pathExists as unknown as Mock).mockImplementation((path: string) => {
      if (path.includes(".stageflow-artifacts.json")) {
        return Promise.resolve(false);
      }
      return Promise.resolve(true);
    });
    (fs.readdir as unknown as Mock).mockResolvedValue([]);
  });

  it("should initialize and run a scan successfully", async () => {
    const results = await scanner.run(mockConfig);

    expect(fs.ensureDir).toHaveBeenCalledWith(mockConfig.resultsDir);

    expect(mocks.mockPageIteratorInstance.loadProvenance).toHaveBeenCalled();
    expect(mocks.mockPageIteratorInstance.iteratePages).toHaveBeenCalled();

    expect(results.jobId).toBe(mockConfig.jobId);
    expect(results.totalPages).toBe(2);
    expect(results.pages).toHaveLength(2);
    expect(results.summary.pagesScanned).toBe(2);

    expect(fs.writeJSON).toHaveBeenCalledWith(
      expect.stringContaining("results.json"),
      expect.anything(),
      expect.anything(),
    );
    expect(
      mocks.mockStorageProviderInstance.upload.mock.calls.length,
    ).toBeGreaterThanOrEqual(2);

    expect(mocks.mockEventPublisherInstance.publishScanCompleted).toHaveBeenCalled();
    expect(mocks.mockEventPublisherInstance.publishPageCompleted).toHaveBeenCalledTimes(
      2,
    );
  });

  it("should handle scan errors gracefully", async () => {
    vi.clearAllMocks();
    (fs.ensureDir as unknown as Mock).mockResolvedValue(undefined);

    mocks.mockPageIteratorInstance.loadProvenance.mockRejectedValueOnce(
      new Error("Provenance missing"),
    );

    await expect(scanner.run(mockConfig)).rejects.toThrow("Provenance missing");

    expect(mocks.mockEventPublisherInstance.publishScanFailed).toHaveBeenCalledWith(
      "Provenance missing",
      expect.anything(),
      expect.objectContaining({
        stageLogPath: expect.stringContaining("stages/scan.mock-scanner.log.json"),
        recipePath: expect.stringContaining("recipes/scan.mock-scanner.json"),
      }),
    );
  });

  it("uploads scanner artifacts when scan succeeds", async () => {
    await scanner.run(mockConfig);

    expect(mocks.mockBrowserManagerInstance.createContext).not.toHaveBeenCalled();
    expect(
      mocks.mockStorageProviderInstance.upload.mock.calls.length,
    ).toBeGreaterThanOrEqual(2);
  });
});
