CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    urls TEXT NOT NULL,
    scanners TEXT,
    baseline_job_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS project_jobs (
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    job_id TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, job_id)
);

-- Durable journal for MinIO baseline mutations. Rows remain until the object
-- mutation and its corresponding project-state transition have both
-- completed, so startup reconciliation can safely replay interrupted work.
CREATE TABLE IF NOT EXISTS baseline_operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL CHECK (kind IN ('promote', 'backfill', 'delete_object', 'delete_project')),
    state TEXT NOT NULL DEFAULT 'object_pending'
        CHECK (state IN ('object_pending', 'commit_pending')),
    project_id TEXT NOT NULL,
    job_id TEXT NOT NULL DEFAULT '',
    previous_job_id TEXT NOT NULL DEFAULT '',
    source_key TEXT NOT NULL DEFAULT '',
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (kind, project_id, job_id)
);

CREATE INDEX IF NOT EXISTS idx_projects_slug ON projects(slug);
CREATE INDEX IF NOT EXISTS idx_project_jobs_job_id ON project_jobs(job_id);
CREATE INDEX IF NOT EXISTS idx_baseline_operations_state
    ON baseline_operations(state, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_baseline_operations_active_promotion
    ON baseline_operations(project_id) WHERE kind = 'promote';
CREATE UNIQUE INDEX IF NOT EXISTS idx_baseline_operations_active_project_delete
    ON baseline_operations(project_id) WHERE kind = 'delete_project';

-- Public job IDs are bearer tokens. A row here hides the job from anonymous
-- reads after the visitor asks to delete it, even if orchestrator metadata
-- has not yet aged out.
CREATE TABLE IF NOT EXISTS deleted_jobs (
    job_id TEXT PRIMARY KEY,
    deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
