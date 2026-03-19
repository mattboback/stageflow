import { createSSEStream } from "$lib/api/sse";
import { createScanReportStore } from "$lib/stores/scan-report.svelte";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("$lib/api/sse", () => ({
	createSSEStream: vi.fn(),
}));

vi.mock("$lib/api/utils", () => ({
	buildApiUrl: (path: string) => path,
}));

type FetchFn = (
	input: RequestInfo | URL,
	init?: RequestInit,
) => Promise<Response>;

const jsonResponse = (payload: unknown, status = 200): Response =>
	new Response(JSON.stringify(payload), {
		status,
		headers: { "content-type": "application/json" },
	});

async function flushPromises(times = 4): Promise<void> {
	for (let i = 0; i < times; i += 1) {
		await Promise.resolve();
	}
}

describe("Scan Report Store", () => {
	let fetchMock: ReturnType<typeof vi.fn<FetchFn>>;

	beforeEach(() => {
		vi.useFakeTimers();
		vi.clearAllMocks();
		fetchMock = vi.fn<FetchFn>();
		vi.stubGlobal("fetch", fetchMock);
		vi.stubGlobal("EventSource", undefined);
		vi.mocked(createSSEStream).mockReturnValue({ close: vi.fn() });
	});

	afterEach(() => {
		vi.useRealTimers();
		vi.unstubAllGlobals();
		vi.restoreAllMocks();
	});

	it("initializes with default values", () => {
		const store = createScanReportStore("job-123");
		expect(store.status).toBe("loading");
		expect(store.job).toBeNull();
		expect(store.report).toBeNull();
		expect(store.screenshots).toEqual([]);
		expect(store.logs).toEqual([]);
		expect(store.error).toBeNull();
	});

	it("polls status when EventSource is unavailable", async () => {
		const store = createScanReportStore("job-123");
		const statusResponse = {
			state: "running",
			progress: { current_page: 5, total_pages: 10 },
		};
		fetchMock.mockImplementation(() =>
			Promise.resolve(jsonResponse(statusResponse)),
		);

		store.start();
		await flushPromises();
		expect(fetchMock).toHaveBeenCalledWith("/api/v1/jobs/job-123");

		await vi.advanceTimersByTimeAsync(5000);
		await flushPromises();
		expect(fetchMock).toHaveBeenCalledTimes(2);
		expect(store.status).toBe("processing");

		store.cleanup();
	});

	it("closes SSE and starts polling after a non-terminal SSE error", async () => {
		const store = createScanReportStore("job-123");
		vi.stubGlobal("EventSource", class EventSourceMock {});

		let onError:
			| ((err: {
					message: string;
					parseError?: boolean;
					terminal?: boolean;
			  }) => void)
			| undefined;
		const close = vi.fn();
		vi.mocked(createSSEStream).mockImplementation((_jobId, callbacks) => {
			onError = callbacks.onError;
			return { close };
		});

		fetchMock.mockImplementation(() =>
			Promise.resolve(jsonResponse({ state: "running" })),
		);

		store.start();
		await flushPromises();
		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(onError).toBeTypeOf("function");

		onError?.({ message: "connection dropped", terminal: false });
		await flushPromises();
		expect(close).toHaveBeenCalledTimes(1);
		expect(fetchMock).toHaveBeenCalledTimes(2);

		await vi.advanceTimersByTimeAsync(5000);
		await flushPromises();
		expect(fetchMock).toHaveBeenCalledTimes(3);

		store.cleanup();
	});
});
