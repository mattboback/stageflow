declare module "lighthouse" {
  export type ReportOutput = string | string[];

  export interface LighthouseFlags {
    logLevel?: "silent" | "error" | "warn" | "info" | "verbose";
    output?: ReportOutput;
    port?: number;
    onlyCategories?: string[] | string;
    onlyAudits?: string[] | string;
    extraHeaders?: Record<string, string>;
  }

  export interface LighthouseConfig {
    extends?: string;
    settings?: Record<string, unknown>;
  }

  export interface LighthouseAudit {
    id?: string;
    title?: string;
    description?: string;
    score?: number | null;
    numericValue?: number;
    displayValue?: string;
    details?: unknown;
  }

  export interface LighthouseCategory {
    id?: string;
    title?: string;
    score?: number | null;
  }

  export interface RunnerResult {
    lhr: {
      finalDisplayedUrl?: string;
      categories: Record<string, LighthouseCategory>;
      audits: Record<string, LighthouseAudit>;
    };
    report?: ReportOutput;
  }

  const lighthouse: (
    url: string,
    flags?: LighthouseFlags,
    config?: LighthouseConfig,
  ) => Promise<RunnerResult>;

  export = lighthouse;
  export default lighthouse;
}
