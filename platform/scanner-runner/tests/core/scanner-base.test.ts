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
    uploadDirectory: vi.fn().mockResolvedValue(0),
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

  validateProvenanceInput(provenance: Provenance): void {
    this.validateProvenance(provenance);
  }

  buildScanSummary(pageResults: PageScanResult[]) {
    return this.buildSummary(pageResults);
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
    (fs.readJSON as unknown as Mock).mockResolvedValue({});
    (fs.stat as unknown as Mock).mockResolvedValue({
      isDirectory: () => false,
      isFile: () => true,
    });
    (fs.pathExists as unknown as Mock).mockImplementation((path: string) => {
      if (path.includes(".stageflow-artifacts.json")) {
        return Promise.resolve(false);
      }
      return Promise.resolve(true);
    });
    (fs.readdir as unknown as Mock).mockResolvedValue([]);
    mocks.mockStorageProviderInstance.uploadDirectory.mockResolvedValue(0);
    mocks.mockStorageProviderInstance.upload.mockResolvedValue(undefined);
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

  it("validates provenance required fields and URL constraints", () => {
    const validProvenance: Provenance = {
      version: "1.0.0",
      job_id: "job-1",
      base_url: "http://localhost:8080",
      pages: [{ id: "page-1", path: "/index.html", url: "http://localhost:8080/index.html" }],
    };

    expect(() => {
      scanner.validateProvenanceInput({ ...validProvenance, version: "" });
    }).toThrow("Provenance version is required");
    expect(() => {
      scanner.validateProvenanceInput({ ...validProvenance, job_id: "" });
    }).toThrow("Provenance job_id is required");
    expect(() => {
      scanner.validateProvenanceInput({ ...validProvenance, pages: [] });
    }).toThrow("Provenance does not contain any pages to scan");
    expect(() => {
      scanner.validateProvenanceInput({
        ...validProvenance,
        base_url: "",
        pages: [{ id: "page-1", path: "/index.html", url: "/index.html" }],
      } as unknown as Provenance);
    }).toThrow("Provenance base_url is required when pages don't have full URLs");

    expect(() => {
      scanner.validateProvenanceInput({
        ...validProvenance,
        base_url: "",
        pages: [{ id: "page-1", path: "/index.html", url: "https://example.com" }],
      } as unknown as Provenance);
    }).not.toThrow();
  });

  it("builds summary with aggregated lighthouse categories", () => {
    const now = new Date().toISOString();
    const summary = scanner.buildScanSummary([
      {
        pageId: "page-1",
        url: "https://example.com",
        success: true,
        issues: [
          {
            id: "issue-1",
            scanner: "mock-scanner",
            severity: "critical",
            category: "accessibility",
            title: "Issue 1",
            description: "desc",
          },
        ],
        durationMs: 100,
        startedAt: now,
        finishedAt: now,
        rawResults: {
          categories: {
            accessibility: { id: "accessibility", title: "Accessibility", score: 0.9 },
            seo: { id: "seo", title: "SEO", score: null },
          },
        },
      },
      {
        pageId: "page-2",
        url: "https://example.com/about",
        success: true,
        issues: [
          {
            id: "issue-2",
            scanner: "mock-scanner",
            severity: "minor",
            category: "seo",
            title: "Issue 2",
            description: "desc",
          },
        ],
        durationMs: 200,
        startedAt: now,
        finishedAt: now,
        rawResults: {
          categories: {
            accessibility: { id: "accessibility", title: "Accessibility", score: 0.7 },
          },
        },
      },
    ]);

    expect(summary.totalIssues).toBe(2);
    expect(summary.bySeverity.critical).toBe(1);
    expect(summary.bySeverity.minor).toBe(1);
    expect(summary.byCategory.accessibility).toBe(1);
    expect(summary.byCategory.seo).toBe(1);
    expect(summary.avgDurationMs).toBe(150);
    expect(summary.lighthouseCategories).toEqual([
      { id: "accessibility", title: "Accessibility", avgScore: 0.8 },
    ]);
  });

  it("wires lifecycle hooks for page callbacks and page errors", async () => {
    const onPageStart = vi.fn().mockResolvedValue(undefined);
    const onPageEnd = vi.fn().mockResolvedValue(undefined);
    const onError = vi.fn().mockResolvedValue(undefined);

    const scannerWithHooks = new MockScanner({
      onPageStart,
      onPageEnd,
      onError,
    });

    mocks.mockPageIteratorInstance.iteratePages.mockImplementationOnce(
      async (provenance: Provenance, scannerFn: PageScanCallback, hooks: PageIteratorCallbacks = {}) => {
        const pageEntry = provenance.pages[0]!;
        await hooks.onPageStart?.(pageEntry, 0, 1);
        await hooks.onPageError?.(new Error("page failed"), pageEntry, 1);
        const result = await scannerFn({ pageEntry } as unknown as ScanContext);
        await hooks.onPageComplete?.(result, 0, 1);
        return [result];
      },
    );

    await scannerWithHooks.run(mockConfig);

    expect(onPageStart).toHaveBeenCalledTimes(1);
    expect(onPageEnd).toHaveBeenCalledTimes(1);
    expect(onError).toHaveBeenCalledWith(expect.any(Error), {
      pageEntry: expect.objectContaining({ id: "page-1" }),
    });
  });

  it("uploads screenshot subdirectories for discovered page folders", async () => {
    (fs.readdir as unknown as Mock).mockResolvedValue([
      { name: "page-1", isDirectory: () => true },
      { name: "ignored-file", isDirectory: () => false },
      { name: "page-2", isDirectory: () => true },
    ]);
    (fs.pathExists as unknown as Mock).mockImplementation((path: string) => {
      if (path.includes(".stageflow-artifacts.json")) {
        return Promise.resolve(false);
      }
      if (path.includes("page-1/screenshots")) {
        return Promise.resolve(true);
      }
      if (path.includes("page-2/screenshots")) {
        return Promise.resolve(false);
      }
      return Promise.resolve(true);
    });
    mocks.mockStorageProviderInstance.uploadDirectory.mockResolvedValue(3);

    await scanner.run(mockConfig);

    expect(mocks.mockStorageProviderInstance.uploadDirectory).toHaveBeenCalledWith(
      mockConfig.storage.bucket,
      "test-job/mock-scanner/page-1/screenshots",
      "/tmp/results/page-1/screenshots",
    );
  });

  it("handles mixed extra artifact manifest entries", async () => {
    (fs.pathExists as unknown as Mock).mockImplementation((path: string) => {
      if (path.includes(".stageflow-artifacts.json")) {
        return Promise.resolve(true);
      }
      if (path.endsWith("missing.txt")) {
        return Promise.resolve(false);
      }
      return Promise.resolve(true);
    });
    (fs.readJSON as unknown as Mock).mockResolvedValue({
      paths: [
        "../outside.txt",
        "missing.txt",
        "broken-stat.txt",
        "socket.bin",
        "dup.txt",
        "dup.txt",
        "dir-ok",
        "dir-fail",
      ],
      files: ["file-fail.txt", 123, "   "],
      directories: ["dir-ok", null],
    });
    (fs.stat as unknown as Mock).mockImplementation((path: string) => {
      if (path.endsWith("broken-stat.txt")) {
        throw new Error("stat failed");
      }
      if (path.endsWith("socket.bin")) {
        return Promise.resolve({ isDirectory: () => false, isFile: () => false });
      }
      if (path.endsWith("dir-ok") || path.endsWith("dir-fail")) {
        return Promise.resolve({ isDirectory: () => true, isFile: () => false });
      }
      return Promise.resolve({ isDirectory: () => false, isFile: () => true });
    });
    mocks.mockStorageProviderInstance.uploadDirectory.mockImplementation(
      (_bucket: string, prefix: string) => {
        if (prefix.endsWith("/dir-fail")) {
          return Promise.reject(new Error("dir upload failed"));
        }
        return Promise.resolve(2);
      },
    );
    mocks.mockStorageProviderInstance.upload.mockImplementation((_bucket: string, key: string) => {
      if (key.endsWith("/file-fail.txt")) {
        return Promise.reject(new Error("file upload failed"));
      }
      return Promise.resolve(undefined);
    });

    await scanner.run(mockConfig);

    expect(mocks.mockStorageProviderInstance.upload).toHaveBeenCalledWith(
      mockConfig.storage.bucket,
      "test-job/mock-scanner/dup.txt",
      "/tmp/results/dup.txt",
    );
    expect(mocks.mockStorageProviderInstance.uploadDirectory).toHaveBeenCalledWith(
      mockConfig.storage.bucket,
      "test-job/mock-scanner/dir-ok",
      "/tmp/results/dir-ok",
    );
  });

  it("ignores extra artifact manifests with no usable paths", async () => {
    (fs.pathExists as unknown as Mock).mockImplementation((path: string) => {
      if (path.includes(".stageflow-artifacts.json")) {
        return Promise.resolve(true);
      }
      return Promise.resolve(true);
    });
    (fs.readJSON as unknown as Mock).mockResolvedValue({
      paths: ["   ", 42],
      files: null,
      directories: "not-an-array",
    });

    await scanner.run(mockConfig);

    expect(mocks.mockStorageProviderInstance.upload).toHaveBeenCalledWith(
      mockConfig.storage.bucket,
      "test-job/mock-scanner/results.json",
      "/tmp/results/results.json",
    );
  });

  it("uploads provenance artifact when SCAN_URLS is set", async () => {
    process.env.SCAN_URLS = "https://example.com";

    try {
      await scanner.run(mockConfig);
    } finally {
      delete process.env.SCAN_URLS;
    }

    expect(mocks.mockStorageProviderInstance.upload).toHaveBeenCalledWith(
      mockConfig.storage.bucket,
      "test-job/provenance.json",
      mockConfig.provenancePath,
      "application/json",
    );
  });
});
