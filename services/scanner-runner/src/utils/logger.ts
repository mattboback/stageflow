import type { ScannerLogger } from "../core/types";

export function createLogger(prefix: string): ScannerLogger {
  const baseMeta: Record<string, unknown> = {};
  if (process.env.JOB_ID) {
    baseMeta.job_id = process.env.JOB_ID;
  }
  if (process.env.REQUEST_ID) {
    baseMeta.request_id = process.env.REQUEST_ID;
  }
  if (process.env.RUN_ID) {
    baseMeta.run_id = process.env.RUN_ID;
  }

  const formatMessage = (
    level: string,
    message: string,
    meta?: Record<string, unknown>,
  ): string => {
    const timestamp = new Date().toISOString();
    const mergedMeta =
      meta && Object.keys(baseMeta).length > 0
        ? { ...baseMeta, ...meta }
        : (meta ?? baseMeta);
    const metaStr =
      Object.keys(mergedMeta).length > 0 ? ` ${JSON.stringify(mergedMeta)}` : "";
    return `[${timestamp}] [${level.toUpperCase()}] [${prefix}] ${message}${metaStr}`;
  };

  return {
    info(message: string, meta?: Record<string, unknown>): void {
      console.log(formatMessage("info", message, meta));
    },
    warn(message: string, meta?: Record<string, unknown>): void {
      console.warn(formatMessage("warn", message, meta));
    },
    error(message: string, meta?: Record<string, unknown>): void {
      console.error(formatMessage("error", message, meta));
    },
    debug(message: string, meta?: Record<string, unknown>): void {
      if (process.env.DEBUG || process.env.NODE_ENV === "development") {
        console.debug(formatMessage("debug", message, meta));
      }
    },
  };
}
