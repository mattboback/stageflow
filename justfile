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
    (cd {{scanner_dir}} && {{bun}} install --frozen-lockfile)

[group('dev'), doc('Local stack: up/down/restart/logs/init (ENV=dev|local)')]
dev CMD='up' ENV='dev' ENDPOINT='http://127.0.0.1:9000':
    #!/usr/bin/env bash
    set -euo pipefail

    cmd="{{CMD}}"
    env="{{ENV}}"
    endpoint="{{ENDPOINT}}"
    root_dir="{{repo_root}}"

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

[group('quality'), doc('Run local CI: lint + typecheck + test')]
ci:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "==> Go build..."
    {{go}} build ./...

    echo "==> Go lint..."
    golangci-lint run

    echo "==> Go test..."
    {{go}} test -race ./...

    echo "==> Frontend CI..."
    (cd {{frontend_dir}} && {{bun}} run ci)

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

[group('prod'), doc('Manage production Quadlets (CMD=install|up|down|restart|logs|ps|health)')]
prod CMD='up':
    #!/usr/bin/env bash
    set -euo pipefail

    cmd="{{CMD}}"

    case "$cmd" in
        install)
            echo "==> Installing Quadlet units..."
            ./scripts/quadlet-install.sh
            ;;
        up)
            echo "==> Starting production stack..."
            ./scripts/quadlet-install.sh
            systemctl --user enable --now stageflow.target
            ;;
        down)
            echo "==> Stopping production stack..."
            systemctl --user stop stageflow.target
            ;;
        restart)
            echo "==> Restarting production stack..."
            systemctl --user restart stageflow.target
            ;;
        logs)
            echo "==> Following production logs..."
            journalctl --user -f \
                -u stageflow-nats.service \
                -u stageflow-minio.service \
                -u stageflow-orchestrator.service \
                -u stageflow-platform-api.service \
                -u stageflow-frontend.service \
                -u stageflow-grafana.service
            ;;
        ps)
            {{podman}} ps --format 'table {{"{{"}}.Names{{"}}"}}\t{{"{{"}}.Status{{"}}"}}\t{{"{{"}}.Ports{{"}}"}}' | grep -E '^(systemd-)?stageflow-' || true
            ;;
        health)
            echo "==> Checking service health..."
            for unit in stageflow-nats.service stageflow-minio.service stageflow-orchestrator.service \
                        stageflow-platform-api.service stageflow-frontend.service stageflow-grafana.service; do
                state="$(systemctl --user is-active "$unit" 2>/dev/null || echo "unknown")"
                echo "$unit: $state"
            done
            ;;
        *)
            echo "CMD must be install, up, down, restart, logs, ps, or health (got: $cmd)" >&2
            exit 2
            ;;
    esac

[group('prod'), doc('Deploy production (MODE=full|quick)')]
deploy MODE='full':
    #!/usr/bin/env bash
    set -euo pipefail

    mode="{{MODE}}"

    case "$mode" in
        full)
            just images
            just prod down
            just prod up
            ;;
        quick)
            just prod down
            just prod up
            ;;
        *)
            echo "MODE must be full or quick (got: $mode)" >&2
            exit 2
            ;;
    esac

[group('run'), doc('Run a service locally (SERVICE=frontend|api|orchestrator MODE=dev|preview)')]
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
        api)
            echo "==> Starting platform-api..."
            (cd platform/api && {{go}} run ./cmd/server)
            ;;
        orchestrator)
            echo "==> Starting orchestrator..."
            (cd platform/orchestrator && {{go}} run ./cmd/orchestrator)
            ;;
        *)
            echo "SERVICE must be frontend, api, or orchestrator (got: $service)" >&2
            exit 2
            ;;
    esac

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
