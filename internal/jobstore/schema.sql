-- Schema v2 is applied only to a fresh PostgreSQL volume.
CREATE TABLE IF NOT EXISTS jobs (
 id TEXT PRIMARY KEY,
 state TEXT NOT NULL CHECK (state IN ('PENDING','EXTRACTING','READY_TO_SCAN','SCANNING','COMPLETING','DONE','FAILED')),
 input_type TEXT NOT NULL CHECK (input_type IN ('zip','urls')),
 input_path TEXT NOT NULL DEFAULT '',
 urls JSONB NOT NULL DEFAULT '[]',
 config_json JSONB NOT NULL DEFAULT '{}',
 pod_id TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
 claimed_at TIMESTAMPTZ,
 extraction_started_at TIMESTAMPTZ,
 scan_started_at TIMESTAMPTZ,
 completed_at TIMESTAMPTZ,
 cleanup_pending BOOLEAN NOT NULL DEFAULT FALSE,
 error TEXT NOT NULL DEFAULT '',
 error_details TEXT NOT NULL DEFAULT '',
 last_stage TEXT NOT NULL DEFAULT '',
 total_pages INTEGER NOT NULL DEFAULT 0 CHECK (total_pages >= 0),
 total_violations INTEGER NOT NULL DEFAULT 0 CHECK (total_violations >= 0),
 report_json_key TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS job_scanners (
 job_id TEXT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
 scanner TEXT NOT NULL,
 state TEXT NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','running','success','partial','failed')),
 attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 2),
 container_id TEXT NOT NULL DEFAULT '',
 pages_scanned INTEGER NOT NULL DEFAULT 0 CHECK (pages_scanned >= 0),
 total_issues INTEGER NOT NULL DEFAULT 0 CHECK (total_issues >= 0),
 critical_issues INTEGER NOT NULL DEFAULT 0 CHECK (critical_issues >= 0),
 serious_issues INTEGER NOT NULL DEFAULT 0 CHECK (serious_issues >= 0),
 moderate_issues INTEGER NOT NULL DEFAULT 0 CHECK (moderate_issues >= 0),
 minor_issues INTEGER NOT NULL DEFAULT 0 CHECK (minor_issues >= 0),
 info_issues INTEGER NOT NULL DEFAULT 0 CHECK (info_issues >= 0),
 timing_ms BIGINT NOT NULL DEFAULT 0 CHECK (timing_ms >= 0),
 error TEXT NOT NULL DEFAULT '',
 results_path TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(job_id,scanner)
);
CREATE INDEX IF NOT EXISTS idx_jobs_queue ON jobs(created_at,id) WHERE state='PENDING' AND claimed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_jobs_retention ON jobs(completed_at) WHERE completed_at IS NOT NULL;
