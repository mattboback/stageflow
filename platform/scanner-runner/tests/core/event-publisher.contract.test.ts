import type { JetStreamClient } from "nats";

import { describe, expect, it } from "vitest";

import type { PageScanResult, ScanResults, ScanTiming } from "../../src/core/types";

import { NatsEventPublisher } from "../../src/core/event-publisher";

function createCapturingPublisher(jobId: string, scannerName: string) {
  const publisher = new NatsEventPublisher(jobId, scannerName);

  const published: { subject: string; envelope: any }[] = [];

  const jetstreamStub: Partial<JetStreamClient> = {
    publish: (subject: string, data: Uint8Array) => {
      const raw = new TextDecoder().decode(data);
      published.push({ subject, envelope: JSON.parse(raw) });
      return Promise.resolve({} as any);
    },
  };

  (publisher as any).jetstream = jetstreamStub;

  return { publisher, published };
}

describe("event contract (scanner-runner -> Go)", () => {
  it("emits scan.page.completed with scanner context and no envelope scanner field", async () => {
    const { publisher, published } = createCapturingPublisher("job-123", "axe");

    const pageResult: PageScanResult = {
      pageId: "page-1",
      url: "http://localhost:8080/index.html",
      success: true,
      issues: [],
      durationMs: 10,
      startedAt: new Date().toISOString(),
      finishedAt: new Date().toISOString(),
    };

    await publisher.publishPageCompleted(pageResult, 1, 3);

    expect(published).toHaveLength(1);
    const msg = published[0]!;
    expect(msg.envelope.scanner).toBeUndefined();

    expect(Object.keys(msg.envelope.payload).sort()).toEqual(
      ["job_id", "scanner_type", "page_id", "page_index", "total_pages"].sort(),
    );

    expect(msg.envelope.payload.scanner_type).toBe("axe");
  });

  it("emits scan.completed with by_severity and snake_case timing keys", async () => {
    const { publisher, published } = createCapturingPublisher("job-123", "axe");

    const results: ScanResults = {
      jobId: "job-123",
      scanner: "axe",
      version: "0.1.0",
      totalPages: 1,
      pages: [
        {
          pageId: "page-1",
          url: "http://localhost:8080/index.html",
          success: true,
          issues: [],
          durationMs: 10,
          startedAt: new Date().toISOString(),
          finishedAt: new Date().toISOString(),
        },
      ],
      summary: {
        totalIssues: 7,
        bySeverity: { critical: 2, serious: 3, moderate: 1, minor: 1, info: 0 },
        byCategory: {},
        pagesScanned: 1,
        pagesFailed: 0,
        pagesWithIssues: 1,
        avgDurationMs: 10,
      },
      startedAt: new Date().toISOString(),
      completedAt: new Date().toISOString(),
      durationMs: 10,
    };

    const timing: ScanTiming = {
      totalMs: 1000,
      pageIterationMs: 600,
      writeResultsMs: 150,
      uploadArtifactsMs: 200,
      publishCompletedMs: 25,
      finalizationMs: 25,
    };

    await publisher.publishScanCompleted(results, timing);

    expect(published).toHaveLength(1);
    const env = published[0]!.envelope;
    expect(env.scanner).toBeUndefined();

    const payload = env.payload;
    expect(payload.summary.by_severity).toBeDefined();
    expect(payload.summary.by_module).toBeUndefined();

    expect(Object.keys(payload.timing).sort()).toEqual(
      [
        "finalization_ms",
        "page_iteration_ms",
        "publish_completed_ms",
        "total_ms",
        "upload_artifacts_ms",
        "write_results_ms",
      ].sort(),
    );
  });
});
