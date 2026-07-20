# StageFlow justfile
# Opinionated, small CLI surface (favor big commands).

# Variables
go := env_var_or_default('GO', 'go')
podman := env_var_or_default('PODMAN', 'podman')
bun := env_var_or_default('BUN', 'bun')
compose_project := env_var_or_default('COMPOSE_PROJECT_NAME', 'stageflow_dev')
repo_root := justfile_directory()

# Paths
web_dir := 'clients/web'
scanner_dir := 'services/scanner-runner'
go_work := 'go.work'

[group('meta'), doc('Show available recipes')]
help:
    @just --list

[group('setup'), doc('One-time-ish setup: Podman network + Go/Bun deps')]
setup:
    #!/usr/bin/env bash
    set -euo pipefail

    project="{{compose_project}}"
    network="${STAGEFLOW_NETWORK_NAME:-${project}_net}"

    echo "==> Ensuring $network network exists..."
    {{podman}} network inspect "$network" >/dev/null 2>&1 || {{podman}} network create "$network"

    just deps

[group('setup'), doc('Install Go/Bun dependencies and default local config')]
deps:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "==> Ensuring scanner config exists..."
    ./infra/scripts/ensure-scanner-config.sh

    echo "==> Generating schema contracts..."
    just generate-contracts

    echo "==> Syncing Go workspace..."
    {{go}} work sync

    echo "==> Installing clients/web dependencies..."
    (cd {{web_dir}} && {{bun}} install --frozen-lockfile)

    echo "==> Installing scanner-runner dependencies..."
    (cd {{scanner_dir}} && PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1 {{bun}} install --frozen-lockfile)

[group('setup'), doc('Check local prerequisites and local-first env hints')]
diagnose:
    @./infra/scripts/diagnose-local-env.sh

[group('demo'), doc('Bootstrap the local stack and print the quickest next steps')]
demo URL='https://example.com':
    #!/usr/bin/env bash
    set -euo pipefail

    url="{{URL}}"
    root_dir="{{repo_root}}"
    current_host="$(hostname -f 2>/dev/null || hostname)"
    demo_vite_api_url="${STAGEFLOW_DEMO_VITE_API_URL:-http://localhost:8080}"
    demo_site_url="${STAGEFLOW_DEMO_VITE_SITE_URL:-http://localhost:3000}"
    demo_cors="${STAGEFLOW_DEMO_CORS_ALLOW_ORIGINS:-http://localhost:3000,http://127.0.0.1:3000,http://localhost:8080,http://localhost:3020,http://127.0.0.1:3020}"
    demo_domain="${STAGEFLOW_DEMO_PUBLIC_DOMAIN:-localhost}"
    demo_grafana_url="${STAGEFLOW_DEMO_GF_SERVER_ROOT_URL:-http://localhost:3001}"

    wait_for_http() {
        local url="$1"
        local label="$2"
        local timeout="${3:-120}"
        local started_at

        started_at="$(date +%s)"

        until curl -fsS "$url" >/dev/null 2>&1; do
            if (( $(date +%s) - started_at >= timeout )); then
                echo "Timed out waiting for ${label} at ${url}" >&2
                exit 1
            fi
            sleep 2
        done

        echo "==> ${label} ready (${url})"
    }

    if [[ -n "${STAGEFLOW_PROTECTED_HOST:-}" && "$current_host" == "$STAGEFLOW_PROTECTED_HOST" && "${STAGEFLOW_ALLOW_VPS_LOCAL_STACKS:-0}" != "1" ]]; then
        echo "Refusing to start the repo-local StageFlow stack on the production VPS." >&2
        echo "Use ${STAGEFLOW_PROD_DEPLOY_DIR:-the external deployment directory} for live operations." >&2
        exit 1
    fi

    if [[ ! -f "$root_dir/.env" ]]; then
        echo "Missing .env: copy .env.example to .env first" >&2
        exit 1
    fi

    echo "==> Diagnosing local prerequisites..."
    just diagnose

    echo "==> Setup..."
    just setup

    echo "==> Building images..."
    VITE_API_URL="$demo_vite_api_url" \
    VITE_SITE_URL="$demo_site_url" \
    STAGEFLOW_PUBLIC_DOMAIN="$demo_domain" \
    PLATFORM_API_CORS_ALLOW_ORIGINS="$demo_cors" \
    GF_SERVER_ROOT_URL="$demo_grafana_url" \
    just images

    echo "==> Restarting MinIO..."
    VITE_API_URL="$demo_vite_api_url" \
    VITE_SITE_URL="$demo_site_url" \
    STAGEFLOW_PUBLIC_DOMAIN="$demo_domain" \
    PLATFORM_API_CORS_ALLOW_ORIGINS="$demo_cors" \
    GF_SERVER_ROOT_URL="$demo_grafana_url" \
    just dev down

    VITE_API_URL="$demo_vite_api_url" \
    VITE_SITE_URL="$demo_site_url" \
    STAGEFLOW_PUBLIC_DOMAIN="$demo_domain" \
    PLATFORM_API_CORS_ALLOW_ORIGINS="$demo_cors" \
    GF_SERVER_ROOT_URL="$demo_grafana_url" \
    just dev up dev http://127.0.0.1:9000 minio

    wait_for_http "http://127.0.0.1:9000/minio/health/live" "MinIO" 120

    echo "==> Initializing MinIO buckets..."
    just dev init

    echo "==> Starting stack..."
    VITE_API_URL="$demo_vite_api_url" \
    VITE_SITE_URL="$demo_site_url" \
    STAGEFLOW_PUBLIC_DOMAIN="$demo_domain" \
    PLATFORM_API_CORS_ALLOW_ORIGINS="$demo_cors" \
    GF_SERVER_ROOT_URL="$demo_grafana_url" \
    just dev up

    wait_for_http "http://127.0.0.1:8080/healthz" "Platform API" 120

    wait_for_http "http://127.0.0.1:3000/" "Frontend" 120

    echo ""
    echo "==> Demo ready"
    echo "UI:      http://localhost:3000"
    echo "API:     http://localhost:8080"
    echo "Grafana: http://localhost:3001"
    echo ""
    echo "Try the web UI, or run:"
    echo "  just cli-install"
    echo "  stageflow scan $url"
    echo ""
    echo "Raw API:"
    echo "curl -sS -X POST http://localhost:8080/api/v1/jobs/urls \\"
    echo "  -H 'content-type: application/json' \\"
    echo "  -d '{\"urls\":[\"$url\"]}'"

[group('dev'), doc('Local stack: up/down/restart/logs/init (ENV=dev|local)')]
dev CMD='up' ENV='dev' ENDPOINT='http://127.0.0.1:9000' SERVICES='':
    #!/usr/bin/env bash
    set -euo pipefail

    cmd="{{CMD}}"
    env="{{ENV}}"
    endpoint="{{ENDPOINT}}"
    services="{{SERVICES}}"
    root_dir="{{repo_root}}"
    current_host="$(hostname -f 2>/dev/null || hostname)"

    if [[ -n "${STAGEFLOW_PROTECTED_HOST:-}" && "$current_host" == "$STAGEFLOW_PROTECTED_HOST" && "${STAGEFLOW_ALLOW_VPS_LOCAL_STACKS:-0}" != "1" ]]; then
        echo "Refusing to run repo-local StageFlow dev stacks on the production VPS." >&2
        echo "Use ${STAGEFLOW_PROD_DEPLOY_DIR:-the external deployment directory} for live operations." >&2
        exit 1
    fi

    project="{{compose_project}}"
    network="${STAGEFLOW_NETWORK_NAME:-${project}_net}"
    export STAGEFLOW_NETWORK_NAME="$network"

    echo "==> Ensuring $network network exists..."
    {{podman}} network inspect "$network" >/dev/null 2>&1 || {{podman}} network create "$network"

    echo "==> Ensuring scanner config exists..."
    ./infra/scripts/ensure-scanner-config.sh

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

    require_job_images() {
        local missing=()
        local image

        for image in \
            "localhost/stageflow/extractor:latest" \
            "localhost/stageflow/scanner-runner:latest"
        do
            if ! {{podman}} image exists "$image"; then
                missing+=("$image")
            fi
        done

        if (( ${#missing[@]} == 0 )); then
            return 0
        fi

        echo "Missing required StageFlow job image(s):" >&2
        printf '  - %s\n' "${missing[@]}" >&2
        echo "Run 'just images' before 'just dev $cmd'." >&2
        exit 1
    }

    case "$cmd" in
        up)
            require_job_images
            echo "==> Starting $env stack..."
            service_args=()
            if [[ -n "${services// }" ]]; then
                read -r -a service_args <<<"$services"
            fi
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" up -d "${service_args[@]}"
            ;;
        down)
            echo "==> Stopping $env stack..."
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" down
            ;;
        restart)
            require_job_images
            echo "==> Restarting $env stack..."
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" down
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" up -d
            ;;
        logs)
            {{podman}} compose -p "$project" "${files[@]}" "${env_args[@]}" logs -f
            ;;
        init)
            echo "==> Initializing MinIO buckets ($endpoint)..."
            declare -A env_overrides=()
            for name in MINIO_ACCESS_KEY MINIO_SECRET_KEY MINIO_ROOT_USER MINIO_ROOT_PASSWORD MINIO_STAGING_RETENTION_DAYS MINIO_ARTIFACT_RETENTION_DAYS MINIO_APPLY_LIFECYCLES MINIO_APP_POLICY MINIO_ALIAS MC_IMAGE PODMAN; do
                if [[ -v "$name" ]]; then
                    env_overrides["$name"]="${!name}"
                fi
            done

            set -a
            [[ -f "$root_dir/.env" ]] && . "$root_dir/.env"
            set +a

            for name in "${!env_overrides[@]}"; do
                export "$name=${env_overrides[$name]}"
            done

            MINIO_ENDPOINT="$endpoint" ./infra/minio/init-buckets.sh
            ;;
        *)
            echo "CMD must be up, down, restart, logs, or init (got: $cmd)" >&2
            exit 2
            ;;
    esac

[group('dev'), doc('Rebuild and recreate selected compose services (ENV=dev|local SERVICES="platform-api orchestrator frontend-react")')]
dev-refresh ENV='local' SERVICES='platform-api orchestrator frontend-react':
    #!/usr/bin/env bash
    set -euo pipefail

    env="{{ENV}}"
    services="{{SERVICES}}"
    root_dir="{{repo_root}}"
    current_host="$(hostname -f 2>/dev/null || hostname)"

    if [[ -n "${STAGEFLOW_PROTECTED_HOST:-}" && "$current_host" == "$STAGEFLOW_PROTECTED_HOST" && "${STAGEFLOW_ALLOW_VPS_LOCAL_STACKS:-0}" != "1" ]]; then
        echo "Refusing to run repo-local StageFlow dev refresh on the production VPS." >&2
        echo "Use ${STAGEFLOW_PROD_DEPLOY_DIR:-the external deployment directory} for live operations." >&2
        exit 1
    fi

    if [[ -z "${services// }" ]]; then
        echo "SERVICES must contain one or more compose service names" >&2
        exit 2
    fi

    project="{{compose_project}}"
    network="${STAGEFLOW_NETWORK_NAME:-${project}_net}"
    export STAGEFLOW_NETWORK_NAME="$network"

    echo "==> Ensuring $network network exists..."
    {{podman}} network inspect "$network" >/dev/null 2>&1 || {{podman}} network create "$network"

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

[group('quality'), doc('Run local CI: lint + test + vuln')]
ci:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "==> Stale-vocabulary and naming drift check..."
    ./devtools/scripts/check-stale-vocab.sh

    echo "==> Internal Markdown links..."
    node devtools/scripts/check-markdown-links.mjs

    echo "==> Generating schema contracts..."
    just generate-contracts

    echo "==> Syncing Go workspace..."
    {{go}} work sync

    echo "==> Ensuring Go lint and vuln tools..."
    {{go}} install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
    {{go}} install golang.org/x/vuln/cmd/govulncheck@v1.1.4
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

    echo "==> Go vulncheck..."
    while IFS= read -r dir; do
        [[ -n "$dir" ]] || continue
        echo "  -> $dir"
        (cd "$dir" && govulncheck ./...)
    done < <(awk '/^[[:space:]]+\.\//{gsub(/^[[:space:]]+/, ""); print}' {{go_work}})

    echo "==> CLI docs..."
    ./devtools/scripts/check-cli-docs-generated.sh

    echo "==> Shell regression tests..."
    just shell-tests

    echo "==> Frontend CI..."
    (cd {{web_dir}} && {{bun}} run ci)

    echo "==> Frontend audit..."
    (cd {{web_dir}} && {{bun}} audit --audit-level=high)

    echo "==> Scanner-runner CI..."
    (cd {{scanner_dir}} && {{bun}} x playwright install chromium)
    (cd {{scanner_dir}} && {{bun}} run ci)

    echo "==> Scanner-runner audit..."
    (cd {{scanner_dir}} && {{bun}} audit --audit-level=moderate --ignore GHSA-8988-4f7v-96qf)

[group('quality'), doc('Generate JSON-schema contract code used by Go and TypeScript builds')]
generate-contracts:
    @./devtools/scripts/generate-contracts.sh

[group('build'), doc('Build all artifacts (clients/web + Go + runner)')]
build:
    #!/usr/bin/env bash
    set -euo pipefail

    just deps

    echo "==> Building Go modules..."
    while IFS= read -r dir; do
        [[ -n "$dir" ]] || continue
        echo "  -> $dir"
        (cd "$dir" && {{go}} build ./...)
    done < <(awk '/^[[:space:]]+\.\//{gsub(/^[[:space:]]+/, ""); print}' {{go_work}})

    echo "==> Building clients/web..."
    (cd {{web_dir}} && {{bun}} run build)

    echo "==> Building scanner-runner..."
    (cd {{scanner_dir}} && {{bun}} run build)

[group('build'), doc('Build container images')]
images:
    @echo "==> Building container images..."
    ./infra/scripts/build-images.sh

[group('deploy'), doc('Production deploys run from the root control plane (this recipe only prints the procedure)')]
deploy:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "StageFlow production deployment is managed from /home/matt/Deployment." >&2
    echo "Run: cd /home/matt/Deployment && just check stageflow && just deploy stageflow" >&2
    exit 1

[group('tools'), doc('Build and install stageflow CLI to ~/.local/bin (no stale binaries)')]
cli-install BIN_DIR='$HOME/.local/bin' BIN_NAME='stageflow':
    @{{repo_root}}/devtools/scripts/install-cli.sh "{{BIN_DIR}}" "{{BIN_NAME}}"

[group('run'), doc('Run a service locally (SERVICE=clients/web|api|orchestrator MODE=dev|preview)')]
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
        clients/web)
            if [[ "$mode" == "preview" ]]; then
                echo "==> Starting clients/web preview server..."
                (cd {{web_dir}} && {{bun}} run preview)
            else
                echo "==> Starting clients/web dev server..."
                (cd {{web_dir}} && {{bun}} run dev)
            fi
            ;;
        api)
            echo "==> Starting platform-api..."
            (cd services/platform-api && {{go}} run ./cmd/server)
            ;;
        orchestrator)
            echo "==> Starting orchestrator..."
            (cd services/orchestrator && {{go}} run ./cmd/orchestrator)
            ;;
        *)
            echo "SERVICE must be clients/web, api, or orchestrator (got: $service)" >&2
            exit 2
            ;;
    esac

[group('quality'), doc('Run repo shell regression tests')]
shell-tests:
    #!/usr/bin/env bash
    set -euo pipefail
    bash devtools/scripts/tests/cli-install.test.sh
    bash devtools/scripts/tests/dev-onboarding.test.sh
    bash devtools/scripts/tests/markdown-links.test.sh
    bash devtools/scripts/tests/stale-vocab.test.sh
    bash infra/minio/provision_test.sh

[group('quality'), doc('Run dead-code analysis for configured TypeScript workspaces')]
dead-code:
    #!/usr/bin/env bash
    set -uo pipefail
    status=0
    (cd {{scanner_dir}} && {{bun}} run find-dead-code) || status=$?
    exit "$status"

[group('quality'), doc('Run the project baseline->promote->diff golden flow against the local overlay')]
project-golden:
    #!/usr/bin/env bash
    set -euo pipefail
    bash qa/e2e/project-scan-golden.sh

[group('cleanup'), doc('Remove artifacts (MODE=all|deep)')]
clean MODE='all':
    #!/usr/bin/env bash
    set -euo pipefail

    mode="{{MODE}}"
    echo "==> Cleaning artifacts..."
    find . -type f \( -name "coverage.out" -o -name "coverage.html" -o -name "*.coverprofile" \) -not -path "*/node_modules/*" -delete
    find . -type d \( -name "dist" -o -name "build" -o -name "coverage" -o -name ".react-router" -o -name ".gocache" -o -name ".impeccable" \) -not -path "*/node_modules/*" -exec rm -rf {} + 2>/dev/null || true
    rm -rf .cache artifacts output .patchright-cli .playwright-cli
    rm -rf libs/contracts/report/generated libs/contracts/provenance/generated libs/contracts/scanner-manifest/generated
    rm -f libs/contracts/scanner-manifest/scanner_manifest.go
    rm -f cli job-status-cli suite-runner scan_result.json stageflow stageflow-cli sbom-*.spdx.json
    rm -f clients/cli/cli clients/cli/report.json clients/cli/stageflow clients/cli/stageflow-cli
    rm -f clients/cli/stageflow_*.tar.gz clients/cli/stageflow_*.zip
    rm -f devtools/ops/job-status-cli/job-status-cli devtools/qa/suite-runner/suite-runner

    if [[ "$mode" == "deep" ]]; then
        echo "==> Removing node_modules..."
        find . -type d -name "node_modules" -not -path "*/node_modules/*/node_modules" -exec rm -rf {} + 2>/dev/null || true
        echo "==> Cleaning Go caches..."
        {{go}} clean -cache -modcache -testcache
    fi
