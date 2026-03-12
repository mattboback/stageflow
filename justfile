# StageFlow justfile
# Opinionated, small CLI surface (favor big commands).

# Variables
go := env_var_or_default('GO', 'go')
podman := env_var_or_default('PODMAN', 'podman')
bun := env_var_or_default('BUN', 'bun')
compose_project := env_var_or_default('COMPOSE_PROJECT_NAME', 'stageflow')
repo_root := justfile_directory()

# Paths
frontend_dir := 'frontend'
scanner_dir := 'platform/scanner-runner'
go_work := 'go.work'

[group('meta'), doc('Show available recipes')]
help:
    @just --list

[group('setup'), doc('One-time-ish setup: Podman network + Go/Bun deps')]
setup:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "==> Ensuring stageflow_net network exists..."
    {{podman}} network inspect stageflow_net >/dev/null 2>&1 || {{podman}} network create stageflow_net

    echo "==> Syncing Go workspace..."
    {{go}} work sync

    echo "==> Installing frontend dependencies..."
    (cd {{frontend_dir}} && {{bun}} install --frozen-lockfile)

    echo "==> Installing scanner-runner dependencies..."
    (cd {{scanner_dir}} && PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 {{bun}} install --frozen-lockfile)

[group('demo'), doc('Bring up the local stack and print a URL-scan demo command')]
demo URL='https://example.com':
    #!/usr/bin/env bash
    set -euo pipefail

    url="{{URL}}"
    root_dir="{{repo_root}}"
    current_host="$(hostname -f 2>/dev/null || hostname)"

    if [[ -n "${STAGEFLOW_PROTECTED_HOST:-}" && "$current_host" == "$STAGEFLOW_PROTECTED_HOST" && "${STAGEFLOW_ALLOW_VPS_LOCAL_STACKS:-0}" != "1" ]]; then
        echo "Refusing to start the repo-local StageFlow stack on the production VPS." >&2
        echo "Use ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} for live operations." >&2
        exit 1
    fi

    if [[ ! -f "$root_dir/.env" ]]; then
        echo "Missing .env: copy .env.example to .env first" >&2
        exit 1
    fi

    echo "==> Setup..."
    just setup

    echo "==> Starting stack..."
    just dev up

    echo "==> Initializing MinIO buckets..."
    just dev init

    echo "==> Building images..."
    just images

    echo ""
    echo "==> Demo ready"
    echo "UI: http://localhost:3000"
    echo ""
    echo "Submit a URL scan:"
    echo "curl -sS -X POST http://localhost:8080/api/v1/jobs/urls \\"
    echo "  -H 'content-type: application/json' \\"
    echo "  -d '{\"urls\":[\"$url\"]}'"

[group('dev'), doc('Local stack: up/down/restart/logs/init (ENV=dev|local)')]
dev CMD='up' ENV='dev' ENDPOINT='http://127.0.0.1:9000':
    #!/usr/bin/env bash
    set -euo pipefail

    cmd="{{CMD}}"
    env="{{ENV}}"
    endpoint="{{ENDPOINT}}"
    root_dir="{{repo_root}}"
    current_host="$(hostname -f 2>/dev/null || hostname)"

    if [[ -n "${STAGEFLOW_PROTECTED_HOST:-}" && "$current_host" == "$STAGEFLOW_PROTECTED_HOST" && "${STAGEFLOW_ALLOW_VPS_LOCAL_STACKS:-0}" != "1" ]]; then
        echo "Refusing to run repo-local StageFlow dev stacks on the production VPS." >&2
        echo "Use ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} for live operations." >&2
        exit 1
    fi

    echo "==> Ensuring stageflow_net network exists..."
    {{podman}} network inspect stageflow_net >/dev/null 2>&1 || {{podman}} network create stageflow_net

    project="{{compose_project}}"
    env_args=()
    [[ -f "$root_dir/.env" ]] && env_args=(--env-file "$root_dir/.env")

    files=()
    case "$env" in
        dev)
            files=(-f "$root_dir/infra/compose/podman-compose.yml" -f "$root_dir/infra/compose/podman-compose.test.yml")
            ;;
        local)
            files=(-f "$root_dir/infra/compose/podman-compose.yml" -f "$root_dir/infra/compose/podman-compose.local.yml")
            ;;
        *)
            echo "ENV must be dev or local (got: $env)" >&2
            exit 2
            ;;
    esac

    case "$cmd" in
        up)
            echo "==> Starting $env stack..."
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" up -d
            ;;
        down)
            echo "==> Stopping $env stack..."
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" down
            ;;
        restart)
            echo "==> Restarting $env stack..."
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" down
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" up -d
            ;;
        logs)
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" logs -f
            ;;
        init)
            echo "==> Initializing MinIO buckets ($endpoint)..."
            set -a
            [[ -f "$root_dir/.env" ]] && . "$root_dir/.env"
            set +a
            MINIO_ENDPOINT="$endpoint" ./infra/minio/init-buckets.sh
            ;;
        *)
            echo "CMD must be up, down, restart, logs, or init (got: $cmd)" >&2
            exit 2
            ;;
    esac

[group('dev'), doc('Rebuild and recreate selected compose services (ENV=dev|local SERVICES=\"platform-api orchestrator frontend\")')]
dev-refresh ENV='local' SERVICES='platform-api orchestrator frontend':
    #!/usr/bin/env bash
    set -euo pipefail

    env="{{ENV}}"
    services="{{SERVICES}}"
    root_dir="{{repo_root}}"
    current_host="$(hostname -f 2>/dev/null || hostname)"

    if [[ -n "${STAGEFLOW_PROTECTED_HOST:-}" && "$current_host" == "$STAGEFLOW_PROTECTED_HOST" && "${STAGEFLOW_ALLOW_VPS_LOCAL_STACKS:-0}" != "1" ]]; then
        echo "Refusing to run repo-local StageFlow dev refresh on the production VPS." >&2
        echo "Use ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} for live operations." >&2
        exit 1
    fi

    if [[ -z "${services// }" ]]; then
        echo "SERVICES must contain one or more compose service names" >&2
        exit 2
    fi

    echo "==> Ensuring stageflow_net network exists..."
    {{podman}} network inspect stageflow_net >/dev/null 2>&1 || {{podman}} network create stageflow_net

    project="{{compose_project}}"
    env_args=()
    [[ -f "$root_dir/.env" ]] && env_args=(--env-file "$root_dir/.env")

    files=()
    case "$env" in
        dev)
            files=(-f "$root_dir/infra/compose/podman-compose.yml" -f "$root_dir/infra/compose/podman-compose.test.yml")
            ;;
        local)
            files=(-f "$root_dir/infra/compose/podman-compose.yml" -f "$root_dir/infra/compose/podman-compose.local.yml")
            ;;
        *)
            echo "ENV must be dev or local (got: $env)" >&2
            exit 2
            ;;
    esac

    read -r -a service_list <<<"$services"

    echo "==> Refreshing $env services: ${service_list[*]}"

    set +e
    refresh_output="$({{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" up -d --build --force-recreate --no-deps "${service_list[@]}" 2>&1)"
    refresh_status=$?
    set -e

    printf '%s\n' "$refresh_output"

    if [[ $refresh_status -eq 0 ]]; then
        exit 0
    fi

    if grep -Eqi 'dependent containers|already exists' <<<"$refresh_output"; then
        echo "==> Podman reported container conflicts. Removing selected services and retrying..."
        {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" rm -sf "${service_list[@]}"
        {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" up -d --build --force-recreate --no-deps "${service_list[@]}"
        exit 0
    fi

    exit "$refresh_status"

[group('staging'), doc('Staging stack via compose: up/down/restart/logs/init/ps')]
staging CMD='up' ENV_FILE='.env.staging' PROJECT='stageflow-staging' NETWORK='stageflow_staging_net' ENDPOINT='http://127.0.0.1:9300':
    #!/usr/bin/env bash
    set -euo pipefail

    cmd="{{CMD}}"
    env_file_input="{{ENV_FILE}}"
    project="{{PROJECT}}"
    fallback_network="{{NETWORK}}"
    endpoint="{{ENDPOINT}}"
    root_dir="{{repo_root}}"
    current_host="$(hostname -f 2>/dev/null || hostname)"

    if [[ -n "${STAGEFLOW_PROTECTED_HOST:-}" && "$current_host" == "$STAGEFLOW_PROTECTED_HOST" && "${STAGEFLOW_ALLOW_VPS_LOCAL_STACKS:-0}" != "1" ]]; then
        echo "Refusing to run repo-local StageFlow staging stacks on the production VPS." >&2
        echo "Use ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} for live operations." >&2
        exit 1
    fi

    if [[ "$env_file_input" = /* ]]; then
        env_file="$env_file_input"
    else
        env_file="$root_dir/$env_file_input"
    fi

    if [[ ! -f "$env_file" ]]; then
        echo "Missing env file: $env_file (copy .env.staging.example first)" >&2
        exit 1
    fi

    files=(-f "$root_dir/infra/compose/podman-compose.yml" -f "$root_dir/infra/compose/podman-compose.staging.yml")
    env_args=(--env-file "$env_file")

    # shellcheck disable=SC1090
    source "$env_file"
    network="${STAGEFLOW_NETWORK_NAME:-$fallback_network}"

    case "$cmd" in
        up)
            echo "==> Ensuring staging network exists: $network"
            {{podman}} network inspect "$network" >/dev/null 2>&1 || {{podman}} network create "$network"

            echo "==> Starting staging stack (project: $project)..."
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" up -d
            ;;
        down)
            echo "==> Stopping staging stack (project: $project)..."
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" down
            ;;
        restart)
            echo "==> Ensuring staging network exists: $network"
            {{podman}} network inspect "$network" >/dev/null 2>&1 || {{podman}} network create "$network"

            echo "==> Restarting staging stack (project: $project)..."
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" down
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" up -d
            ;;
        logs)
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" logs -f
            ;;
        init)
            echo "==> Initializing staging MinIO buckets ($endpoint)..."
            set -a
            # shellcheck disable=SC1090
            source "$env_file"
            set +a
            MINIO_ENDPOINT="$endpoint" ./infra/minio/init-buckets.sh
            ;;
        ps)
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" ps
            ;;
        *)
            echo "CMD must be up, down, restart, logs, init, or ps (got: $cmd)" >&2
            exit 2
            ;;
    esac

[group('quality'), doc('Run local CI: lint + typecheck + test + Storybook')]
ci:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "==> Ensuring Go lint tool..."
    {{go}} install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0
    export PATH="$({{go}} env GOPATH)/bin:$PATH"

    echo "==> Go build..."
    while IFS= read -r dir; do
        [[ -n "$dir" ]] || continue
        echo "  -> $dir"
        (cd "$dir" && {{go}} build ./...)
    done < <(awk '/^[[:space:]]+\.\//{gsub(/^[[:space:]]+/, ""); print}' {{go_work}})

    echo "==> Go lint..."
    while IFS= read -r dir; do
        [[ -n "$dir" ]] || continue
        echo "  -> $dir"
        (cd "$dir" && golangci-lint run --allow-parallel-runners)
    done < <(awk '/^[[:space:]]+\.\//{gsub(/^[[:space:]]+/, ""); print}' {{go_work}})

    echo "==> Go test..."
    while IFS= read -r dir; do
        [[ -n "$dir" ]] || continue
        echo "  -> $dir"
        (cd "$dir" && {{go}} test -race ./...)
    done < <(awk '/^[[:space:]]+\.\//{gsub(/^[[:space:]]+/, ""); print}' {{go_work}})

    echo "==> CLI docs..."
    {{go}} run ./tools/stageflow-cli docs --out-dir docs/generated/cli
    git diff --exit-code docs/generated/cli

    echo "==> Shell regression tests..."
    bash scripts/tests/cli-install.test.sh

    echo "==> Frontend CI..."
    (cd {{frontend_dir}} && {{bun}} run ci)

    echo "==> Frontend Storybook browser setup..."
    (cd {{frontend_dir}} && {{bun}} x playwright install chromium)
    echo "==> Frontend Storybook tests..."
    (cd {{frontend_dir}} && {{bun}} run test-storybook)

    echo "==> Frontend audit..."
    (cd {{frontend_dir}} && {{bun}} audit --audit-level=high)

    echo "==> Scanner-runner CI..."
    (cd {{scanner_dir}} && {{bun}} x playwright install chromium)
    (cd {{scanner_dir}} && {{bun}} run ci)

    echo "==> Scanner-runner audit..."
    (cd {{scanner_dir}} && {{bun}} audit --audit-level=high)

[group('build'), doc('Build all artifacts (Go + frontend + runner)')]
build:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "==> Building Go modules..."
    while IFS= read -r dir; do
        [[ -n "$dir" ]] || continue
        echo "  -> $dir"
        (cd "$dir" && {{go}} build ./...)
    done < <(awk '/^[[:space:]]+\.\//{gsub(/^[[:space:]]+/, ""); print}' {{go_work}})

    echo "==> Building frontend..."
    (cd {{frontend_dir}} && {{bun}} run build)

    echo "==> Building scanner-runner..."
    (cd {{scanner_dir}} && {{bun}} run build)

[group('build'), doc('Build container images')]
images:
    @echo "==> Building container images..."
    ./scripts/build-images.sh

[group('tools'), doc('Build and install stageflow CLI to ~/.local/bin (no stale binaries)')]
cli-install BIN_DIR='$HOME/.local/bin' BIN_NAME='stageflow':
    #!/usr/bin/env bash
    set -euo pipefail

    bin_dir="{{BIN_DIR}}"
    bin_name="{{BIN_NAME}}"
    dest="${bin_dir}/${bin_name}"

    mkdir -p "$bin_dir"

    tmp="$(mktemp -t stageflow-cli.XXXXXX)"
    trap 'rm -f "$tmp"' EXIT

    echo "==> Building StageFlow CLI..."
    version="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
    commit="$(git rev-parse --short HEAD 2>/dev/null || echo "")"
    build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    ldflags="-s -w -X main.version=${version} -X main.commit=${commit} -X main.date=${build_date}"
    (cd tools/stageflow-cli && {{go}} build -trimpath -ldflags "$ldflags" -o "$tmp" .)

    echo "==> Installing: $dest"
    install -m 0755 "$tmp" "$dest"

    resolved="$(command -v "$bin_name" || true)"
    if [[ -z "$resolved" ]]; then
        echo "Installed '$dest', but '$bin_dir' is not on PATH. Add it to PATH, then run '${bin_name} version'." >&2
        exit 1
    fi
    if [[ "$resolved" != "$dest" ]]; then
        echo "Installed '$dest', but '$bin_name' does not resolve to the installed binary (got '$resolved'). Reorder PATH or remove the stale binary, then run '${bin_name} version'." >&2
        exit 1
    fi

    echo "==> Installed and available on PATH as '$bin_name'."
    "$resolved" version

[group('prod'), doc('Production is owned by external control plane; this recipe intentionally stops old repo-local usage')]
prod CMD='up':
    #!/usr/bin/env bash
    set -euo pipefail

    echo "Production control for StageFlow lives at ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment}." >&2
    echo "Use one of these commands instead:" >&2
    echo "  cd ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} && just deploy stageflow" >&2
    echo "  cd ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} && just restart stageflow" >&2
    echo "  cd ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} && just logs stageflow" >&2
    echo "  cd ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} && just stop stageflow" >&2
    echo "  cd ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} && just health" >&2
    exit 1

[group('prod'), doc('Production deployment is owned by external control plane; this recipe intentionally stops old repo-local usage')]
deploy MODE='full':
    #!/usr/bin/env bash
    set -euo pipefail

    echo "Production deployment for StageFlow lives at ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment}." >&2
    echo "Use one of these commands instead:" >&2
    echo "  cd ${STAGEFLOW_PROD_DEPLOY_DIR:-/home/matt/Deployment} && just deploy stageflow" >&2
    exit 1

[group('run'), doc('Run a service locally (SERVICE=frontend|storybook|api|orchestrator MODE=dev|preview)')]
run SERVICE MODE='dev':
    #!/usr/bin/env bash
    set -euo pipefail

    service="{{SERVICE}}"
    mode="{{MODE}}"
    root_dir="{{repo_root}}"

    if [[ -f "$root_dir/.env" ]]; then
        set -a
        # shellcheck disable=SC1091
        source "$root_dir/.env"
        set +a
    fi

    case "$service" in
        frontend)
            if [[ "$mode" == "preview" ]]; then
                echo "==> Starting frontend preview server..."
                (cd {{frontend_dir}} && {{bun}} run preview)
            else
                echo "==> Starting frontend dev server..."
                (cd {{frontend_dir}} && {{bun}} run dev)
            fi
            ;;
        storybook)
            echo "==> Starting frontend Storybook..."
            (cd {{frontend_dir}} && {{bun}} run storybook)
            ;;
        api)
            echo "==> Starting platform-api..."
            (cd platform/api && {{go}} run ./cmd/server)
            ;;
        orchestrator)
            echo "==> Starting orchestrator..."
            (cd platform/orchestrator && {{go}} run ./cmd/orchestrator)
            ;;
        *)
            echo "SERVICE must be frontend, storybook, api, or orchestrator (got: $service)" >&2
            exit 2
            ;;
    esac

[group('quality'), doc('Run frontend Storybook interaction + accessibility tests')]
storybook-test:
    #!/usr/bin/env bash
    set -euo pipefail
    (cd {{frontend_dir}} && {{bun}} x playwright install chromium)
    (cd {{frontend_dir}} && {{bun}} run test-storybook)

[group('quality'), doc('Run repo shell regression tests')]
shell-tests:
    #!/usr/bin/env bash
    set -euo pipefail
    bash scripts/tests/cli-install.test.sh

[group('cleanup'), doc('Remove artifacts (MODE=all|deep)')]
clean MODE='all':
    #!/usr/bin/env bash
    set -euo pipefail

    mode="{{MODE}}"
    echo "==> Cleaning artifacts..."
    find . -type f \( -name "coverage.out" -o -name "coverage.html" -o -name "*.coverprofile" \) -not -path "*/node_modules/*" -delete
    find . -type d \( -name "dist" -o -name "build" -o -name "coverage" -o -name ".svelte-kit" -o -name ".gocache" \) -not -path "*/node_modules/*" -exec rm -rf {} + 2>/dev/null || true
    rm -f tools/job-status-cli/job-status-cli tools/suite-runner/suite-runner

    if [[ "$mode" == "deep" ]]; then
        echo "==> Removing node_modules..."
        find . -type d -name "node_modules" -not -path "*/node_modules/*/node_modules" -exec rm -rf {} + 2>/dev/null || true
        echo "==> Cleaning Go caches..."
        {{go}} clean -cache -modcache -testcache
    fi
