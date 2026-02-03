/**
 * Config Loader
 *
 * Loads scanner configuration from environment variables.
 */

import { join } from "node:path";

import type {
  BrowserConfig,
  MessagingConfig,
  ScannerConfig,
  StorageConfig,
} from "./types";

export interface ConfigLoaderOptions {
  scannerName: string;
  defaults?: Partial<ScannerConfig>;
}

const DEFAULT_DATA_DIR = join(process.cwd(), "data");

export function loadConfigFromEnv(options: ConfigLoaderOptions): ScannerConfig {
  const { scannerName, defaults } = options;

  const jobId = requireEnv("JOB_ID");
  const requestId = process.env.REQUEST_ID;
  const runId = process.env.RUN_ID;
  const dataDir = process.env.SCANNER_DATA_DIR ?? DEFAULT_DATA_DIR;

  const browser = loadBrowserConfig();
  const storage = loadStorageConfig();
  const messaging = loadMessagingConfig();

  const config: ScannerConfig = {
    ...defaults,
    jobId,
    requestId: requestId?.trim() ?? undefined,
    runId: runId?.trim() ?? undefined,
    provenancePath:
      process.env.PROVENANCE_PATH ??
      defaults?.provenancePath ??
      join(dataDir, "provenance.json"),
    resultsDir:
      process.env.RESULTS_DIR ?? defaults?.resultsDir ?? join(dataDir, "results"),
    scannerName,
    concurrency: getEnvInt("SCAN_CONCURRENCY", defaults?.concurrency ?? 4),
    maxRetries: getEnvInt("MAX_RETRIES", defaults?.maxRetries ?? 3),
    browser,
    storage,
    messaging,
  };

  const scannerOptions = loadScannerOptionsFromEnv();
  if (scannerOptions) {
    config.options = scannerOptions;
  }

  return config;
}

function loadBrowserConfig(): BrowserConfig {
  return {
    headless: getEnvBool("BROWSER_HEADLESS", true),
    args: [
      "--no-sandbox",
      "--disable-setuid-sandbox",
      "--disable-dev-shm-usage",
      "--disable-gpu",
      ...(process.env.BROWSER_ARGS?.split(",").filter(Boolean) ?? []),
    ],
    defaultViewport: {
      width: getEnvInt("VIEWPORT_WIDTH", 1280),
      height: getEnvInt("VIEWPORT_HEIGHT", 720),
    },
    deviceScaleFactor: getEnvNumber("DEVICE_SCALE_FACTOR", 2),
    defaultTimeout: getEnvInt("DEFAULT_TIMEOUT", 30_000),
    pageLoadTimeout: getEnvInt("PAGE_LOAD_TIMEOUT", 15_000),
  };
}

function loadStorageConfig(): StorageConfig {
  const isDev = process.env.NODE_ENV !== "production";

  const endpoint = process.env.MINIO_ENDPOINT;
  const accessKey = process.env.MINIO_ACCESS_KEY ?? process.env.MINIO_ROOT_USER;
  const secretKey = process.env.MINIO_SECRET_KEY ?? process.env.MINIO_ROOT_PASSWORD;
  const bucket = process.env.MINIO_ARTIFACT_BUCKET;

  // In production, require all storage credentials
  if (!isDev) {
    if (!endpoint) {
      throw new Error("MINIO_ENDPOINT is required in production");
    }
    if (!accessKey) {
      throw new Error("MINIO_ACCESS_KEY is required in production");
    }
    if (!secretKey) {
      throw new Error("MINIO_SECRET_KEY is required in production");
    }
    if (!bucket) {
      throw new Error("MINIO_ARTIFACT_BUCKET is required in production");
    }
  }

  // Development-only fallbacks with warning
  if (isDev && (!endpoint || !accessKey || !secretKey)) {
    console.warn(
      "[config] Using development storage defaults. Set MINIO_* env vars for production.",
    );
  }

  return {
    endpoint: endpoint ?? "localhost:9000",
    accessKey: accessKey ?? "minioadmin",
    secretKey: secretKey ?? "minioadmin",
    useSSL: getEnvBool("MINIO_USE_SSL", false),
    bucket: bucket ?? "scanner-artifacts",
  };
}

function loadMessagingConfig(): MessagingConfig {
  const isDev = process.env.NODE_ENV !== "production";
  const url = process.env.NATS_URL;

  if (!isDev && !url) {
    throw new Error("NATS_URL is required in production");
  }

  if (isDev && !url) {
    console.warn("[config] Using development NATS URL. Set NATS_URL for production.");
  }

  return {
    url: url ?? "nats://localhost:4222",
    subjects: {
      pageCompleted:
        process.env.NATS_SUBJECT_PAGE_COMPLETED ?? "scan.events.page.completed",
      scanCompleted: process.env.NATS_SUBJECT_SCAN_COMPLETED ?? "scan.events.completed",
      scanFailed: process.env.NATS_SUBJECT_SCAN_FAILED ?? "scan.events.failed",
    },
  };
}

function requireEnv(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Required environment variable ${name} is not set`);
  }
  return value;
}

function getEnvBool(name: string, defaultValue: boolean): boolean {
  const raw = process.env[name];
  if (!raw) {
    return defaultValue;
  }
  const normalized = raw.trim().toLowerCase();
  if (["0", "false", "no", "off"].includes(normalized)) {
    return false;
  }
  if (["1", "true", "yes", "on"].includes(normalized)) {
    return true;
  }
  return defaultValue;
}

function getEnvNumber(name: string, defaultValue: number): number {
  const raw = process.env[name];
  if (!raw) {
    return defaultValue;
  }
  const n = Number(raw);
  return Number.isFinite(n) ? n : defaultValue;
}

function getEnvInt(name: string, defaultValue: number): number {
  const raw = process.env[name];
  if (!raw) {
    return defaultValue;
  }
  const n = parseInt(raw, 10);
  return Number.isFinite(n) && n >= 0 ? n : defaultValue;
}

function loadScannerOptionsFromEnv(): Record<string, unknown> | undefined {
  const raw = process.env.SCANNER_OPTIONS;
  if (!raw) {
    return undefined;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    throw new Error(`Failed to parse SCANNER_OPTIONS as JSON: ${message}`);
  }

  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("SCANNER_OPTIONS must be a JSON object");
  }

  return parsed as Record<string, unknown>;
}

export function validateConfig(config: ScannerConfig): void {
  const errors: string[] = [];

  if (!config.jobId) {
    errors.push("jobId is required");
  }

  if (!config.provenancePath) {
    errors.push("provenancePath is required");
  }

  if (!config.resultsDir) {
    errors.push("resultsDir is required");
  }

  if (!config.scannerName) {
    errors.push("scannerName is required");
  }

  if (config.concurrency < 1) {
    errors.push("concurrency must be at least 1");
  }

  if (config.maxRetries < 1) {
    errors.push("maxRetries must be at least 1");
  }

  if (errors.length === 0) {
    return;
  }

  throw new Error(`Invalid scanner configuration:\n  - ${errors.join("\n  - ")}`);
}
