import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import type { ScannerConfig } from "../../src/core/types";

import {
	loadConfigFromEnv,
	validateConfig,
} from "../../src/core/config-loader";

const ORIGINAL_ENV = process.env;

function resetEnv(): void {
	process.env = { ...ORIGINAL_ENV };
	delete process.env.NODE_ENV;
}

function setRequiredRuntimeEnv(): void {
	process.env.MINIO_ENDPOINT = "minio.example.com:9000";
	process.env.MINIO_ACCESS_KEY = "access";
	process.env.MINIO_SECRET_KEY = "secret";
	process.env.MINIO_ARTIFACT_BUCKET = "artifacts";
	process.env.NATS_URL = "nats://nats.example.com:4222";
}

describe("config-loader", () => {
	beforeEach(() => {
		resetEnv();
	});

	afterEach(() => {
		resetEnv();
		vi.restoreAllMocks();
	});

	it("throws when required env vars are missing", () => {
		delete process.env.JOB_ID;
		expect(() => loadConfigFromEnv({ scannerName: "axe" })).toThrow(
			"Required environment variable JOB_ID is not set",
		);
	});

	it("parses scanner options from JSON and applies defaults", () => {
		setRequiredRuntimeEnv();
		process.env.JOB_ID = "job-123";
		process.env.SCAN_CONCURRENCY = "2";
		process.env.MAX_RETRIES = "5";
		process.env.SCANNER_OPTIONS = JSON.stringify({ locale: "en-US" });

		const config = loadConfigFromEnv({
			scannerName: "axe",
			defaults: {
				resultsDir: "/tmp/results",
				provenancePath: "/tmp/provenance.json",
			},
		});

		expect(config.jobId).toBe("job-123");
		expect(config.concurrency).toBe(2);
		expect(config.maxRetries).toBe(5);
		expect(config.options).toEqual({ locale: "en-US" });
		expect(config.storage.bucket).toBe("artifacts");
	});

	describe("runtime environment validation", () => {
		it("throws when MINIO_ENDPOINT is missing", () => {
			process.env.JOB_ID = "job-123";
			setRequiredRuntimeEnv();
			delete process.env.MINIO_ENDPOINT;

			expect(() => loadConfigFromEnv({ scannerName: "axe" })).toThrow(
				"Required environment variable MINIO_ENDPOINT is not set",
			);
		});

		it("throws when MINIO access key aliases are missing", () => {
			process.env.JOB_ID = "job-123";
			setRequiredRuntimeEnv();
			delete process.env.MINIO_ACCESS_KEY;
			delete process.env.MINIO_ROOT_USER;

			expect(() => loadConfigFromEnv({ scannerName: "axe" })).toThrow(
				"Required environment variable not set (expected one of: MINIO_ACCESS_KEY, MINIO_ROOT_USER)",
			);
		});

		it("throws when NATS_URL is missing", () => {
			process.env.JOB_ID = "job-123";
			setRequiredRuntimeEnv();
			delete process.env.NATS_URL;

			expect(() => loadConfigFromEnv({ scannerName: "axe" })).toThrow(
				"Required environment variable NATS_URL is not set",
			);
		});

		it("succeeds when all required env vars are set", () => {
			process.env.JOB_ID = "job-123";
			setRequiredRuntimeEnv();

			const config = loadConfigFromEnv({ scannerName: "axe" });

			expect(config.storage.endpoint).toBe("minio.example.com:9000");
			expect(config.storage.accessKey).toBe("access");
			expect(config.messaging.url).toBe("nats://nats.example.com:4222");
		});
	});

	describe("required aliases", () => {
		it("accepts MINIO_ROOT_USER and MINIO_ROOT_PASSWORD aliases", () => {
			process.env.JOB_ID = "job-123";
			process.env.MINIO_ENDPOINT = "minio.example.com:9000";
			process.env.MINIO_ROOT_USER = "root-access";
			process.env.MINIO_ROOT_PASSWORD = "root-secret";
			delete process.env.MINIO_ACCESS_KEY;
			delete process.env.MINIO_SECRET_KEY;
			process.env.MINIO_ARTIFACT_BUCKET = "artifacts";
			process.env.NATS_URL = "nats://nats.example.com:4222";

			const config = loadConfigFromEnv({ scannerName: "axe" });
			expect(config.storage.accessKey).toBe("root-access");
			expect(config.storage.secretKey).toBe("root-secret");
		});
	});

	it("throws when scanner options JSON is invalid", () => {
		setRequiredRuntimeEnv();
		process.env.JOB_ID = "job-123";
		process.env.SCANNER_OPTIONS = "{nope";

		expect(() => loadConfigFromEnv({ scannerName: "axe" })).toThrow(
			"Failed to parse SCANNER_OPTIONS",
		);
	});

	it("throws when scanner options are not an object", () => {
		setRequiredRuntimeEnv();
		process.env.JOB_ID = "job-123";
		process.env.SCANNER_OPTIONS = JSON.stringify(["bad"]);

		expect(() => loadConfigFromEnv({ scannerName: "axe" })).toThrow(
			"SCANNER_OPTIONS must be a JSON object",
		);
	});

	it("validateConfig reports invalid settings", () => {
		const config = {
			jobId: "",
			provenancePath: "",
			resultsDir: "",
			scannerName: "",
			concurrency: 0,
			maxRetries: 0,
			browser: {
				headless: true,
				args: [],
				defaultViewport: { width: 1280, height: 720 },
				deviceScaleFactor: 2,
				defaultTimeout: 1000,
				pageLoadTimeout: 1000,
			},
			storage: {
				endpoint: "localhost:9000",
				accessKey: "minio",
				secretKey: "minio",
				useSSL: false,
				bucket: "bucket",
			},
			messaging: {
				url: "nats://localhost:4222",
				subjects: {
					pageCompleted: "page.completed",
					scanCompleted: "scan.completed",
					scanFailed: "scan.failed",
				},
			},
		} satisfies ScannerConfig;

		expect(() => {
			validateConfig(config);
		}).toThrow("Invalid scanner configuration");
	});
});
