# StageFlow justfile
# Opinionated, small CLI surface (favor big commands).

# Load local settings once; explicit environment variables take precedence.
set dotenv-load := true
set dotenv-path := x'${ENV_FILE:-.env}'

# Variables
go := env_var_or_default('GO', 'go')
podman := env_var_or_default('PODMAN', 'podman')
bun := env_var_or_default('BUN', 'bun')
compose_project := env_var_or_default('COMPOSE_PROJECT_NAME', 'stageflow_dev')
repo_root := justfile_directory()

# Paths
web_dir := 'clients/web'
scanner_dir := 'services/scanner-runner'

[group('meta'), doc('Show available recipes')]
help:
    @just --list

[group('setup'), doc('One-time-ish setup: Podman networks + Go/Bun deps')]
setup:
    #!/usr/bin/env bash
    set -euo pipefail

    project="{{compose_project}}"
    network="${STAGEFLOW_NETWORK_NAME:-${project}_net}"
    jobs_network="${STAGEFLOW_JOBS_NETWORK_NAME:-${project}_jobs}"

    echo "==> Ensuring $network and isolated $jobs_network networks exist..."
    {{podman}} network inspect "$network" >/dev/null 2>&1 || {{podman}} network create "$network"
    {{podman}} network inspect "$jobs_network" >/dev/null 2>&1 || {{podman}} network create --opt isolate=strict "$jobs_network"

    just deps

[group('setup'), doc('Install Go/Bun dependencies')]
deps:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "==> Downloading Go modules..."
    {{go}} mod download

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
    demo_vite_api_url="${STAGEFLOW_DEMO_VITE_API_URL:-${VITE_API_URL:-http://localhost:${PLATFORM_API_PORT:-8080}}}"
    demo_site_url="${STAGEFLOW_DEMO_VITE_SITE_URL:-${VITE_SITE_URL:-http://localhost:${FRONTEND_PORT:-3000}}}"
    demo_cors="${STAGEFLOW_DEMO_CORS_ALLOW_ORIGINS:-${PLATFORM_API_CORS_ALLOW_ORIGINS:-http://localhost:${FRONTEND_PORT:-3000},http://127.0.0.1:${FRONTEND_PORT:-3000},http://localhost:${PLATFORM_API_PORT:-8080},http://localhost:3020,http://127.0.0.1:3020}}"
    demo_domain="${STAGEFLOW_DEMO_PUBLIC_DOMAIN:-${STAGEFLOW_PUBLIC_DOMAIN:-localhost}}"

    api_endpoint="http://127.0.0.1:${PLATFORM_API_PORT:-8080}"
    frontend_endpoint="http://127.0.0.1:${FRONTEND_PORT:-3000}"

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

    env_file="${ENV_FILE:-$root_dir/.env}"
    if [[ ! -f "$env_file" ]]; then
        echo "Missing $env_file: copy .env.example to your environment file first" >&2
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
    just images

    echo "==> Restarting local stack..."
    VITE_API_URL="$demo_vite_api_url" \
    VITE_SITE_URL="$demo_site_url" \
    STAGEFLOW_PUBLIC_DOMAIN="$demo_domain" \
    PLATFORM_API_CORS_ALLOW_ORIGINS="$demo_cors" \
    just dev down

    echo "==> Starting stack..."
    VITE_API_URL="$demo_vite_api_url" \
    VITE_SITE_URL="$demo_site_url" \
    STAGEFLOW_PUBLIC_DOMAIN="$demo_domain" \
    PLATFORM_API_CORS_ALLOW_ORIGINS="$demo_cors" \
    just dev up

    wait_for_http "$api_endpoint/healthz" "Platform API" 120

    wait_for_http "$frontend_endpoint/" "Frontend" 120

    echo ""
    echo "==> Demo ready"
    echo "UI:      $demo_site_url"
    echo "API:     $demo_vite_api_url"
    echo ""
    echo "Try the web UI, or run:"
    echo "  just cli-install"
    echo "  stageflow scan $url"
    echo ""
    echo "Raw API:"
    echo "curl -sS -X POST $demo_vite_api_url/api/v1/jobs/urls \\"
    echo "  -H 'content-type: application/json' \\"
    echo "  -d '{\"urls\":[\"$url\"]}'"

[group('dev'), doc('Local stack: up/down/restart/logs (dev/local aliases; settings from .env)')]
dev CMD='up' ENV='dev' SERVICES='':
    #!/usr/bin/env bash
    set -euo pipefail

    cmd="{{CMD}}"
    env="{{ENV}}"
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
    jobs_network="${STAGEFLOW_JOBS_NETWORK_NAME:-${project}_jobs}"
    export STAGEFLOW_NETWORK_NAME="$network"
    export STAGEFLOW_JOBS_NETWORK_NAME="$jobs_network"

    echo "==> Ensuring $network and isolated $jobs_network networks exist..."
    {{podman}} network inspect "$network" >/dev/null 2>&1 || {{podman}} network create "$network"
    {{podman}} network inspect "$jobs_network" >/dev/null 2>&1 || {{podman}} network create --opt isolate=strict "$jobs_network"

    env_file="${ENV_FILE:-$root_dir/.env}"
    env_args=()
    [[ -f "$env_file" ]] && env_args=(--env-file "$env_file")

    files=(-f "$root_dir/infra/compose/podman-compose.yml")
    case "$env" in
        dev|local)
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
            "localhost/stageflow/extractor:${IMAGE_TAG:-latest}" \
            "localhost/stageflow/scanner-runner:${IMAGE_TAG:-latest}"
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
        *)
            echo "CMD must be up, down, restart, or logs (got: $cmd)" >&2
            exit 2
            ;;
    esac

[group('dev'), doc('Rebuild selected services (dev/local aliases; settings from .env)')]
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
    jobs_network="${STAGEFLOW_JOBS_NETWORK_NAME:-${project}_jobs}"
    export STAGEFLOW_NETWORK_NAME="$network"
    export STAGEFLOW_JOBS_NETWORK_NAME="$jobs_network"

    echo "==> Ensuring $network and isolated $jobs_network networks exist..."
    {{podman}} network inspect "$network" >/dev/null 2>&1 || {{podman}} network create "$network"
    {{podman}} network inspect "$jobs_network" >/dev/null 2>&1 || {{podman}} network create --opt isolate=strict "$jobs_network"

    env_file="${ENV_FILE:-$root_dir/.env}"
    env_args=()
    [[ -f "$env_file" ]] && env_args=(--env-file "$env_file")

    files=(-f "$root_dir/infra/compose/podman-compose.yml")
    case "$env" in
        dev|local)
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

[group('quality'), doc('Run local CI: contracts, lint, tests, vulnerabilities, secrets, dead code, and browser e2e')]
ci:
    #!/usr/bin/env bash
    set -euo pipefail

    echo "==> Internal Markdown links..."
    node devtools/scripts/check-markdown-links.mjs

    echo "==> Generated contracts..."
    ./devtools/scripts/check-contracts-generated.sh

    echo "==> Contract fixtures and TypeScript packages..."
    (cd libs/contracts/outcome && {{bun}} install --frozen-lockfile && {{bun}} run validate:fixtures)
    for dir in libs/contracts/report libs/contracts/provenance libs/contracts/scanner-manifest; do
        (cd "$dir" && {{bun}} install --frozen-lockfile && {{bun}} run build && {{bun}} run typecheck)
    done

    echo "==> Ensuring pinned quality tools..."
    {{go}} install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
    {{go}} install golang.org/x/vuln/cmd/govulncheck@v1.1.4
    {{go}} install github.com/zricethezav/gitleaks/v8@v8.30.1
    {{go}} install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
    export PATH="$({{go}} env GOPATH)/bin:$PATH"

    echo "==> Workflow lint..."
    command -v shellcheck >/dev/null
    actionlint -color -shellcheck="$(command -v shellcheck)"

    echo "==> Full Git history secrets scan..."
    if [[ "$(git rev-parse --is-shallow-repository)" == "true" ]]; then
        echo "Full-history secret scanning requires a complete checkout; run git fetch --unshallow." >&2
        exit 1
    fi
    gitleaks git --redact --no-banner --log-opts=--all

    echo "==> Pre-commit hooks..."
    python3 -m venv .cache/pre-commit
    .cache/pre-commit/bin/python -m pip install --disable-pip-version-check 'pre-commit==4.6.0'
    .cache/pre-commit/bin/pre-commit run --all-files

    echo "==> Go build..."
    {{go}} build ./...

    echo "==> Go lint..."
    # Analysis cache positions contain absolute paths; isolate worktree caches.
    export GOLANGCI_LINT_CACHE="{{repo_root}}/.cache/golangci-lint"
    golangci-lint run --allow-parallel-runners ./...

    echo "==> Go test..."
    {{go}} test -race ./...

    echo "==> Go vulncheck..."
    govulncheck ./...

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

    echo "==> Production dead-code gates..."
    just dead-code

    echo "==> Frontend browser e2e..."
    just web-e2e

[group('release'), doc('Publish CLI release VERSION (vX.Y.Z) by hand: tag HEAD first and push the tag')]
cli-release VERSION:
    ./devtools/scripts/build-cli-release.sh {{VERSION}}
    gh release create {{VERSION}} --verify-tag --generate-notes --title "StageFlow CLI {{VERSION}}" artifacts/cli-release/{{VERSION}}/*

[group('quality'), doc('Regenerate the committed JSON-schema contract code (Go and TypeScript)')]
generate-contracts:
    @./devtools/scripts/generate-contracts.sh

[group('build'), doc('Build all artifacts (clients/web + Go + runner)')]
build:
    #!/usr/bin/env bash
    set -euo pipefail

    just deps

    echo "==> Building Go..."
    {{go}} build ./...

    echo "==> Building clients/web..."
    (cd {{web_dir}} && {{bun}} run build)

    echo "==> Building scanner-runner..."
    (cd {{scanner_dir}} && {{bun}} run build)

[group('build'), doc('Build container images')]
images:
    @echo "==> Building container images..."
    ./infra/scripts/build-images.sh

[group('deploy'), doc('Production deploys run from an external control plane (this recipe only prints the procedure)')]
deploy:
    #!/usr/bin/env bash
    set -euo pipefail
    control_plane="${STAGEFLOW_PROD_DEPLOY_DIR:-}"
    echo "StageFlow production deploys are not driven from this repository." >&2
    if [[ -n "$control_plane" ]]; then
        echo "Run: cd $control_plane && just check stageflow && just deploy stageflow" >&2
    else
        echo "Set STAGEFLOW_PROD_DEPLOY_DIR to your control-plane directory (see .env.example)," >&2
        echo "then run: just check stageflow && just deploy stageflow from there." >&2
        echo "Self-hosting without a control plane: see docs/self-hosting.md." >&2
    fi
    exit 1

[group('tools'), doc('Build and install stageflow CLI to ~/.local/bin (no stale binaries)')]
cli-install BIN_DIR='$HOME/.local/bin' BIN_NAME='stageflow':
    @{{repo_root}}/devtools/scripts/install-cli.sh "{{BIN_DIR}}" "{{BIN_NAME}}"

[group('tools'), doc('Regenerate the committed sample-site baseline/current demo')]
demo-fixture API='http://localhost:8080':
    @API="{{API}}" {{repo_root}}/devtools/scripts/build-demo-fixture.sh

[group('run'), doc('Run a service locally (SERVICE=clients/web|api|orchestrator MODE=dev|preview)')]
run SERVICE MODE='dev':
    #!/usr/bin/env bash
    set -euo pipefail

    service="{{SERVICE}}"
    mode="{{MODE}}"

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
    bash devtools/scripts/tests/build-images.test.sh
    bash devtools/scripts/tests/dev-onboarding.test.sh
    bash devtools/scripts/tests/quadlet-install.test.sh
    bash devtools/scripts/tests/markdown-links.test.sh
    bash devtools/scripts/tests/secrets-gate.test.sh
    node --test devtools/scripts/tests/check-knip.test.mjs
    bash qa/e2e/scan-helpers.test.sh

[group('quality'), doc('Run production Knip and Go deadcode with reviewed temporary allowances')]
dead-code:
    #!/usr/bin/env bash
    set -euo pipefail
    {{go}} install golang.org/x/tools/cmd/deadcode@v0.47.0
    export PATH="$({{go}} env GOPATH)/bin:$PATH"
    node devtools/scripts/check-knip.mjs
    ./devtools/scripts/check-go-deadcode.py --self-test
    ./devtools/scripts/check-go-deadcode.py

[group('quality'), doc('Run browser e2e against a fresh production web build')]
web-e2e:
    #!/usr/bin/env bash
    set -euo pipefail
    cd {{web_dir}}
    {{bun}} x playwright install chromium
    CI=1 PORT=4173 {{bun}} run test:e2e

[group('quality'), doc('Run ZIP sample scans and compare their regression diff against goldens')]
scan-diff-golden:
    #!/usr/bin/env bash
    set -euo pipefail
    bash qa/e2e/scan-diff-golden.sh

[group('cleanup'), doc('Remove artifacts (MODE=all|deep)')]
clean MODE='all':
    #!/usr/bin/env bash
    set -euo pipefail

    mode="{{MODE}}"
    echo "==> Cleaning artifacts..."
    find . -type f \( -name "coverage.out" -o -name "coverage.html" -o -name "*.coverprofile" \) -not -path "*/node_modules/*" -delete
    find . -type d \( -name "dist" -o -name "build" -o -name "coverage" -o -name ".react-router" -o -name ".gocache" -o -name ".impeccable" \) -not -path "*/node_modules/*" -exec rm -rf {} + 2>/dev/null || true
    rm -rf .cache artifacts output .patchright-cli .playwright-cli
    rm -f cli scan_result.json stageflow stageflow-cli sbom-*.spdx.json
    rm -f clients/cli/cli clients/cli/report.json clients/cli/stageflow clients/cli/stageflow-cli

    if [[ "$mode" == "deep" ]]; then
        echo "==> Removing node_modules..."
        find . -type d -name "node_modules" -not -path "*/node_modules/*/node_modules" -exec rm -rf {} + 2>/dev/null || true
        echo "==> Cleaning Go caches..."
        {{go}} clean -cache -modcache -testcache
    fi

[group('quality'), doc('Verify isolated rootless Podman runtime contracts')]
test-podman-live:
    {{go}} test -tags=podmanlive ./services/orchestrator/internal/podman -run '^TestLiveContract_'
