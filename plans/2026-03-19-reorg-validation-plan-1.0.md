# StageFlow Project Reorganization Validation Plan

This plan outlines the steps required to systematically validate the recent repository restructuring (moving from `platform/`, `packages/`, `tools/` to `services/`, `libs/`, `clients/`, etc.). We will verify that the project can be built, tested, and run end-to-end after the changes.

## Phase 1: Artifact Cleanup and Local Build Verification
- [ ] 1.1 Clean stale artifacts left over from the old structure (e.g., `stageflow`, `stageflow-cli` at the root).
- [ ] 1.2 Run a deep clean (`just clean deep`) to wipe out old Go caches, SvelteKit build directories, and `node_modules`.
- [ ] 1.3 Re-run `just setup` to ensure all Go workspace dependencies and Bun dependencies are correctly installed in their new locations.
- [ ] 1.4 Run `just build` to confirm that all Go binaries, the SvelteKit frontend, and the Playwright scanner-runner compile successfully with the updated paths.

## Phase 2: Comprehensive Testing (CI)
- [ ] 2.1 Run the full CI pipeline locally (`just ci`). This includes:
  - Go linting (`golangci-lint`) across all `libs/` and `services/`.
  - Go unit and integration tests (`go test -race ./...`).
  - Frontend (`clients/web`) tests and type checks.
  - Scanner Runner (`services/scanner-runner`) tests.
  - CLI shell regression tests (`devtools/scripts/tests/cli-install.test.sh`).

## Phase 3: Container Image Validation
- [ ] 3.1 Review and update `infra/scripts/build-images.sh` if it hardcodes paths to `platform/` or `packages/`.
- [ ] 3.2 Build all local container images via `just images`.
- [ ] 3.3 Verify the images are correctly tagged and present in the local Podman registry (`podman images | grep stageflow`).
- [ ] 3.4 Specifically verify the `orchestrator`, `platform-api`, `archive-extractor`, and `scanner-runner` images built successfully using the rewritten Dockerfiles.

## Phase 4: End-to-End Stack Validation (Local Dev)
- [ ] 4.1 Update any remaining hardcoded paths in `infra/compose/podman-compose.yml` and `infra/compose/podman-compose.local.yml` (e.g., build context paths or volume mounts pointing to `platform/`).
- [ ] 4.2 Start the local development stack (`just dev up`).
- [ ] 4.3 Verify all containers transition to a healthy state (`podman ps` and `just dev logs`).
- [ ] 4.4 Initialize MinIO buckets (`just dev init`).
- [ ] 4.5 Execute a full end-to-end scan using the local API:
      `curl -sS -X POST http://localhost:8080/api/v1/jobs/urls -H 'content-type: application/json' -d '{"urls":["https://example.com"]}'`
- [ ] 4.6 Poll the API or check the UI at `http://localhost:3000` to ensure the scan completes successfully, confirming that the orchestrator, API, and scanner-runner are communicating correctly.

## Phase 5: Production Deployment Drift Guard (Review Only)
- [ ] 5.1 Review the Quadlet template definitions in `infra/quadlets/templates/` to ensure they reference the correct new container image names and bind mount paths.
- [ ] 5.2 Document any required changes to the external deployment repository (`~/Deployment/stageflow`) so the operator knows what to update when deploying this refactored code.