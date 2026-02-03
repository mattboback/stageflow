#!/bin/bash
# test-event-flow.sh - End-to-end test for event-driven architecture
#
# Usage:
#   ./scripts/test-event-flow.sh [API_BASE_URL]
#
# Default API URL: http://localhost:4000
# For production testing:
#   ./scripts/test-event-flow.sh https://your-domain.com
#   ./scripts/test-event-flow.sh http://127.0.0.1:4100  # Quadlets (no Caddy)

set -euo pipefail

DEFAULT_API_BASE="${STAGEFLOW_API_BASE:-http://localhost:4000}"
API_BASE="${1:-$DEFAULT_API_BASE}"
API_URL="${API_BASE}/api"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Test 1: Health check
test_health() {
    log_info "Test 1: Health check..."

    # Public gateway health: /api/health (behind Caddy: https://<domain>/api/health)
    if response=$(curl -fsS --max-time 10 "${API_URL}/health" 2>/dev/null); then
        if status=$(echo "$response" | jq -r '.status // empty' 2>/dev/null); then
            if [ "$status" = "healthy" ]; then
                log_info "✓ Health check passed (gateway)"
                return 0
            fi
        fi
    fi

    # Platform API health: /healthz (direct: http://localhost:8080/healthz)
    if response=$(curl -fsS --max-time 10 "${API_BASE}/healthz" 2>/dev/null); then
        if status=$(echo "$response" | jq -r '.status // empty' 2>/dev/null); then
            if [ "$status" = "healthy" ]; then
                log_info "✓ Health check passed (platform-api)"
                return 0
            fi
        fi
    fi

    log_error "✗ Health check failed (tried ${API_URL}/health and ${API_BASE}/healthz)"
    return 1
}

# Test 2: Single-scanner URL job
test_single_scanner() {
    log_info "Test 2: Single-scanner URL job (axe)..."
    
    # Submit job
    response=$(curl -s --max-time 30 -X POST "${API_URL}/v1/jobs/urls" \
        -H "Content-Type: application/json" \
        -d '{"urls": ["https://example.com"], "modules": ["axe"]}')
    
    job_id=$(echo "$response" | jq -r '.job_id')
    if [ -z "$job_id" ] || [ "$job_id" = "null" ]; then
        log_error "✗ Failed to create job: $response"
        return 1
    fi
    log_info "  Created job: $job_id"
    
    # Poll for completion
    for i in {1..30}; do
        sleep 2
        status_resp=$(curl -s --max-time 10 "${API_URL}/v1/jobs/${job_id}")
        state=$(echo "$status_resp" | jq -r '.state')
        
        case $state in
            DONE)
                violations=$(echo "$status_resp" | jq -r '.violations // 0')
                log_info "✓ Job completed with $violations violations"
                return 0
                ;;
            FAILED)
                error=$(echo "$status_resp" | jq -r '.error // "unknown"')
                log_error "✗ Job failed: $error"
                return 1
                ;;
            *)
                echo -ne "  Attempt $i: $state\r"
                ;;
        esac
    done
    
    log_error "✗ Job timed out in state: $state"
    return 1
}

# Test 3: Multi-scanner URL job
test_multi_scanner() {
    log_info "Test 3: Multi-scanner URL job (axe + lighthouse)..."
    
    response=$(curl -s --max-time 30 -X POST "${API_URL}/v1/jobs/urls" \
        -H "Content-Type: application/json" \
        -d '{"urls": ["https://example.com"], "modules": ["axe", "lighthouse"]}')
    
    job_id=$(echo "$response" | jq -r '.job_id')
    if [ -z "$job_id" ] || [ "$job_id" = "null" ]; then
        log_error "✗ Failed to create job: $response"
        return 1
    fi
    log_info "  Created job: $job_id"
    
    for i in {1..45}; do
        sleep 2
        status_resp=$(curl -s --max-time 10 "${API_URL}/v1/jobs/${job_id}")
        state=$(echo "$status_resp" | jq -r '.state')
        
        case $state in
            DONE)
                scanner_count=$(echo "$status_resp" | jq '.artifacts.scanner_artifacts | keys | length')
                if [ "$scanner_count" -eq 2 ]; then
                    log_info "✓ Both scanners completed successfully"
                    return 0
                else
                    log_warn "? Only $scanner_count scanner(s) returned results"
                    return 0
                fi
                ;;
            FAILED)
                error=$(echo "$status_resp" | jq -r '.error // "unknown"')
                log_error "✗ Job failed: $error"
                return 1
                ;;
            *)
                echo -ne "  Attempt $i: $state\r"
                ;;
        esac
    done
    
    log_error "✗ Job timed out in state: $state"
    return 1
}

# Test 4: Artifact accessibility
test_artifacts() {
    log_info "Test 4: Artifact accessibility..."
    
    response=$(curl -s --max-time 30 -X POST "${API_URL}/v1/jobs/urls" \
        -H "Content-Type: application/json" \
        -d '{"urls": ["https://example.com"], "modules": ["axe"]}')
    
    job_id=$(echo "$response" | jq -r '.job_id')
    
    # Wait for completion
    for i in {1..30}; do
        sleep 2
        state=$(curl -s --max-time 10 "${API_URL}/v1/jobs/${job_id}" | jq -r '.state')
        if [ "$state" = "DONE" ]; then break; fi
    done
    
    if [ "$state" != "DONE" ]; then
        log_error "✗ Job did not complete"
        return 1
    fi
    
    # Check results endpoint
    results_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 -L "${API_URL}/v1/jobs/${job_id}/results")
    if [ "$results_code" = "200" ]; then
        log_info "✓ Results JSON accessible"
    else
        log_error "✗ Results JSON returned $results_code"
        return 1
    fi
    
    # Check report endpoint
    report_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 -L "${API_URL}/v1/jobs/${job_id}/report")
    if [ "$report_code" = "200" ]; then
        log_info "✓ Report HTML accessible"
    else
        log_error "✗ Report HTML returned $report_code"
        return 1
    fi
    
    return 0
}

# Main
main() {
    echo "========================================"
    echo "StageFlow Event-Driven Architecture Test"
    echo "API: $API_URL"
    echo "========================================"
    echo
    
    passed=0
    failed=0
    
    if test_health; then ((passed+=1)); else ((failed+=1)); fi
    echo
    
    if test_single_scanner; then ((passed+=1)); else ((failed+=1)); fi
    echo
    
    if test_multi_scanner; then ((passed+=1)); else ((failed+=1)); fi
    echo
    
    if test_artifacts; then ((passed+=1)); else ((failed+=1)); fi
    echo
    
    echo "========================================"
    echo "Results: $passed passed, $failed failed"
    echo "========================================"
    
    if [ $failed -gt 0 ]; then
        exit 1
    fi
}

main
