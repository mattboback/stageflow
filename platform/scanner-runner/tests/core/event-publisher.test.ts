import type { JetStreamClient, NatsConnection } from "nats";

import { beforeEach, describe, expect, it, vi } from "vitest";

import type { PageScanResult, ScanResults } from "../../src/core/types";

import { NatsEventPublisher } from "../../src/core/event-publisher";

const mocks = vi.hoisted(() => {
  const connect = vi.fn();

  const encode = (input: string) => new TextEncoder().encode(input);
  const StringCodec = () => ({ encode });

  return { connect, StringCodec };
});

vi.mock("nats", () => ({
  connect: mocks.connect,
  StringCodec: mocks.StringCodec,
}));

function createLogger() {
  return {
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    debug: vi.fn(),
  };
}

function decodeEnvelope(data: Uint8Array): unknown {
  return JSON.parse(new TextDecoder().decode(data)) as unknown;
}

describe("NatsEventPublisher", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("throws for non-suppressed publishes when not connected", async () => {
    const logger = createLogger();
    const publisher = new NatsEventPublisher("job-1", "axe", undefined, logger);

    const pageResult: PageScanResult = {
      pageId: "page-1",
      url: "https://example.com",
      success: true,
      issues: [],
      durationMs: 10,
      startedAt: new Date().toISOString(),
      finishedAt: new Date().toISOString(),
    };

    await expect(publisher.publishPageCompleted(pageResult, 0, 1)).rejects.toThrow(
      "Not connected to NATS",
    );
  });

  it("suppresses scan.failed when not connected", async () => {
    const logger = createLogger();
    const publisher = new NatsEventPublisher("job-1", "axe", undefined, logger);

    await expect(publisher.publishScanFailed("boom")).resolves.toBeUndefined();
    expect(logger.warn).toHaveBeenCalledWith("Cannot publish - not connected to NATS", {
      event: "scan.failed",
    });
  });

  it("connects, publishes envelopes (with correlation ids), and closes cleanly", async () => {
    const published: { subject: string; env: any }[] = [];

    const jetstream: Pick<JetStreamClient, "publish"> = {
      publish: (subject: string, data: Uint8Array) => {
        published.push({ subject, env: decodeEnvelope(data) as any });
        return Promise.resolve({} as any);
      },
    };

    const connection: Pick<NatsConnection, "jetstream" | "drain" | "close"> = {
      jetstream: () => jetstream as JetStreamClient,
      drain: vi.fn().mockResolvedValue(undefined),
      close: vi.fn().mockResolvedValue(undefined),
    };

    mocks.connect.mockResolvedValue(connection);

    const logger = createLogger();
    const publisher = new NatsEventPublisher(
      "job-123",
      "axe",
      undefined,
      logger,
      { requestId: "req-1", runId: "run-2" },
    );

    await publisher.connect("nats://localhost:4222");

    const pageResult: PageScanResult = {
      pageId: "page-1",
      url: "https://example.com",
      success: true,
      issues: [],
      durationMs: 10,
      startedAt: new Date().toISOString(),
      finishedAt: new Date().toISOString(),
    };

    const results: ScanResults = {
      jobId: "job-123",
      scanner: "axe",
      version: "0.1.0",
      totalPages: 1,
      pages: [pageResult],
      summary: {
        totalIssues: 0,
        bySeverity: { critical: 0, serious: 0, moderate: 0, minor: 0, info: 0 },
        byCategory: {},
        pagesScanned: 1,
        pagesFailed: 0,
        pagesWithIssues: 0,
        avgDurationMs: 10,
      },
      startedAt: new Date().toISOString(),
      completedAt: new Date().toISOString(),
      durationMs: 10,
    };

    await publisher.publishPageCompleted(pageResult, 0, 1);
    await publisher.publishScanCompleted(results, undefined, {
      stageLogPath: "job-123/axe/stage.log",
      recipePath: "job-123/axe/recipe.json",
    });

    expect(mocks.connect).toHaveBeenCalledWith(
      expect.objectContaining({
        servers: "nats://localhost:4222",
        name: "scanner-axe-job-123",
      }),
    );

    expect(published).toHaveLength(2);

    const pageEnv = published.find((m) => m.env.event === "scan.page.completed")!.env;
    expect(pageEnv.job_id).toBe("job-123");
    expect(pageEnv.request_id).toBe("req-1");
    expect(pageEnv.run_id).toBe("run-2");
    expect(pageEnv.producer).toBe("axe");
    expect(pageEnv.payload.page_id).toBe("page-1");

    const completedEnv = published.find((m) => m.env.event === "scan.completed")!.env;
    expect(completedEnv.payload.stage_log_path).toBe("job-123/axe/stage.log");
    expect(completedEnv.payload.recipe_path).toBe("job-123/axe/recipe.json");

    await publisher.close();

    const internal = publisher as unknown as { connection: unknown; jetstream: unknown };
    expect(internal.connection).toBeNull();
    expect(internal.jetstream).toBeNull();
    expect(connection.drain).toHaveBeenCalledTimes(1);
  });

  it("falls back to close when drain fails", async () => {
    const connection: Pick<NatsConnection, "jetstream" | "drain" | "close"> = {
      jetstream: () => ({ publish: vi.fn() }) as unknown as JetStreamClient,
      drain: vi.fn().mockRejectedValue(new Error("drain failed")),
      close: vi.fn().mockResolvedValue(undefined),
    };
    mocks.connect.mockResolvedValue(connection);

    const publisher = new NatsEventPublisher("job-1", "axe", undefined, createLogger());
    await publisher.connect("nats://localhost:4222");
    await publisher.close();

    expect(connection.close).toHaveBeenCalledTimes(1);
  });
});
