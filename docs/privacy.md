# Hosted Demo Data Handling

This page describes the public no-account demo at [stageflow.org](https://stageflow.org). Self-hosted operators control their own storage, access, and retention settings.

## What a Scan May Store

Depending on the scan type and enabled scanners, StageFlow may temporarily store:

- Submitted static-site ZIP archives.
- Page URLs, titles, DOM snippets, response metadata, and scanner findings.
- Full-page screenshots and per-finding image evidence.
- Generated HTML and JSON reports.
- Scanner logs and timing/error information.
- Playwright storage state supplied for an authenticated scan.

Do not submit confidential builds, private customer data, production credentials, or sensitive authenticated targets to the hosted demo. Use a controlled self-hosted deployment for those workflows.

## Retention and Access

The hosted demo configures both staging uploads and ordinary completed scan artifacts to expire after **24 hours**. Object-store lifecycle deletion is asynchronous, so an object may disappear shortly after that window rather than at an exact second. Reports explicitly promoted as project baselines are copied to private persistent storage and remain there until replaced or the project is deleted.

Operational metadata has a separate lifecycle. The durable job record includes the submitted URL, selected scanner configuration, state, and timing information and is not automatically deleted in this release. The event audit trail defaults to 30-day retention. The 24-hour promise applies to uploaded files and ordinary generated object-store artifacts, not durable database records or explicitly promoted project baselines. Do not put credentials or sensitive values in submitted URLs or scanner configuration.

Buckets are private. The Platform API returns short-lived signed artifact URLs rather than exposing objects anonymously. On the no-account demo, a job ID is an unguessable bearer-style reference: anyone who receives the job or report URL may retrieve its status and report until the data expires. A signed artifact URL may likewise be used by anyone who receives it until that signature expires.

StageFlow does not currently provide an immediate user-triggered deletion endpoint. If immediate deletion is a requirement, do not use the hosted demo.

## Authentication Data

Storage-state authentication uploads can contain session cookies or tokens. StageFlow stores them under the job prefix, passes only an artifact reference through events, writes hydrated files with restricted permissions inside the scanner workspace, and removes the local hydrated copy during scanner cleanup. The object-store copy remains subject to the 24-hour lifecycle.

Form recipes should reference credentials through explicitly allow-listed environment variables on a trusted self-hosted scanner. The no-account demo is not a credential vault.

## Self-Hosted Configuration

Self-hosted deployments configure retention with:

- `MINIO_STAGING_RETENTION_DAYS`
- `MINIO_ARTIFACT_RETENTION_DAYS`

Both default to one day. Authentication storage state uses the artifact lifecycle and is deleted on terminal job cleanup when possible; the lifecycle is a fallback expiry. See the [configuration reference](reference/configuration.md) and [self-hosting guide](self-hosting.md). Operators are responsible for their own database-record retention, privacy notice, backups, access controls, and applicable legal obligations.
