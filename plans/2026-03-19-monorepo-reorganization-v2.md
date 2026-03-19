# StageFlow Repository Reorganization Plan

## Objective

To reorganize the StageFlow repository structure into a standard, clean monorepo layout (apps, services, libs, tools, qa) to improve discoverability, clarify architectural boundaries, and consolidate fragmented domains without breaking existing CI/CD or deployment workflows.

## Implementation Plan

### Phase 1: Directory Restructuring

- [x] Task 1. Create new top-level directories (`apps/`, `tools/`).
  Rationale: Establishes the new root-level structure for user-facing applications and internal developer tooling.
- [x] Task 2. Move frontend web application from `clients/web` to `apps/web` using `git mv`.
  Rationale: Clarifies that the web UI is a primary user-facing application rather than just a "client".
- [x] Task 3. Move the official CLI tool from `clients/cli/stageflow` to `apps/cli` using `git mv`.
  Rationale: Elevates the CLI to a primary application and removes unnecessary deep nesting.
- [x] Task 4. Move internal job status CLI from `devtools/ops/job-status-cli` to `tools/job-status-cli` using `git mv`.
  Rationale: Consolidates internal operational tools into a dedicated `tools/` directory.
- [x] Task 5. Move QA suite runner from `devtools/qa/suite-runner` to `qa/suite-runner` using `git mv`.
  Rationale: Consolidates all testing and quality assurance tools into the existing `qa/` domain.
- [x] Task 6. Move utility scripts from `devtools/scripts` to `tools/scripts` using `git mv`.
  Rationale: Centralizes developer and CI scripts alongside other internal tooling.
- [x] Task 7. Remove empty `clients/` and `devtools/` directories.
  Rationale: Cleans up legacy directory structures to prevent confusion.

### Phase 2: Configuration Updates

- [x] Task 8. Update `go.work` to reflect new module paths (`./apps/cli`, `./tools/job-status-cli`, `./qa/suite-runner`).
  Rationale: Ensures the Go workspace can locate and compile all Go modules in their new locations.
- [x] Task 9. Update `justfile` variables and paths (e.g., `web_dir := 'apps/web'`).
  Rationale: Ensures all developer workflows (setup, dev, build, ci) function correctly with the new structure.
- [x] Task 10. Update `justfile` Go CLI build paths in `cli-install` and `ci` recipes.
  Rationale: Fixes local installation and documentation generation for the newly moved CLI application.
- [x] Task 11. Update shell regression test paths in `justfile` to point to `tools/scripts/tests/cli-install.test.sh`.
  Rationale: Ensures CI shell tests can be located and executed.

### Phase 3: Infrastructure and CI Updates

- [x] Task 12. Update `infra/scripts/build-images.sh` to reference `apps/web` and `apps/cli` instead of `clients/`.
  Rationale: Fixes the container image build process which relies on specific source paths.
- [x] Task 13. Update Dockerfiles in `infra/` (if any hardcode source paths) to match the new directory structure.
  Rationale: Ensures container builds can successfully copy source code from the correct locations.
- [~] Task 14. Search and update any remaining hardcoded paths in documentation (`README.md`, `AGENTS.md`, etc.).
  Rationale: Prevents developer confusion by keeping documentation aligned with the actual repository structure.

## Verification Criteria

- [ ] `go work sync` completes successfully without missing module errors.
- [ ] `just setup` completes successfully, installing all Node/Go dependencies.
- [ ] `just ci` passes entirely (linting, Go tests, Playwright tests, Storybook tests).
- [ ] `just build` successfully compiles all Go binaries, the SvelteKit app, and the scanner-runner.
- [ ] `just images` successfully builds all required container images without path errors.
- [ ] `just dev up` successfully starts the local development stack.

## Potential Risks and Mitigations

1. **Broken Container Builds**
   *Risk:* The production deployment process (managed externally) or local image builds might rely on the old `clients/` paths.
   *Mitigation:* Thoroughly search `infra/` for hardcoded paths. The verification step `just images` will confirm if local builds work. We must ensure no external deployment scripts are hardcoded to `clients/web` (as per `AGENTS.md`, deployment is managed externally, so we assume it builds from the Dockerfiles provided in this repo).
2. **Git History Loss**
   *Risk:* Moving files manually instead of using `git mv` could sever file history, making it harder to track changes.
   *Mitigation:* Explicitly use `git mv` for all directory restructuring tasks.
3. **Dangling References in Documentation**
   *Risk:* Onboarding guides or internal docs might still point developers to `clients/web`.
   *Mitigation:* Perform a global text search for `clients/` and `devtools/` across all markdown files and update them.

## Alternative Approaches

1. **Minimal Reorganization (Flattening only)**: Instead of creating `apps/` and `tools/`, simply move `clients/cli/stageflow` up to `clients/cli` and leave `devtools/` as is. 
   *Trade-offs:* Less disruptive, but fails to solve the ambiguous naming of "clients" and keeps domains fragmented.
2. **Domain-Driven Structure**: Organize by feature domain (e.g., `scanning/`, `reporting/`, `core/`) rather than technical role (apps vs services).
   *Trade-offs:* Highly disruptive, requires massive refactoring of imports and module names, and is generally overkill for a repository of this size. The proposed technical-role structure is standard and well-understood.