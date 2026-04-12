#!/usr/bin/env bash
# =============================================================================
# kcli Build Validation Script
# =============================================================================
# This script validates the kcli codebase compiles, passes vet/lint checks,
# runs all unit tests, and meets the PRD's non-functional requirements.
#
# Prerequisites:
#   - Go 1.25+ installed
#   - golangci-lint installed (optional, for lint checks)
#   - goreleaser installed (optional, for release validation)
#
# Usage:
#   chmod +x scripts/validate-build.sh
#   ./scripts/validate-build.sh           # Run all checks
#   ./scripts/validate-build.sh --quick   # Compile + vet only
#   ./scripts/validate-build.sh --full    # All checks + benchmarks + race detector
# =============================================================================

set -euo pipefail

# ── Colors ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

PASS=0
FAIL=0
WARN=0
SKIP=0

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

MODE="${1:-default}"
LOG_FILE="$ROOT_DIR/build-validation-$(date +%Y%m%d-%H%M%S).log"

# ── Helpers ──────────────────────────────────────────────────────────────────

header() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

check_pass() {
    echo -e "  ${GREEN}✓${NC} $1"
    PASS=$((PASS + 1))
}

check_fail() {
    echo -e "  ${RED}✗${NC} $1"
    echo -e "    ${RED}→ $2${NC}"
    FAIL=$((FAIL + 1))
}

check_warn() {
    echo -e "  ${YELLOW}⚠${NC} $1"
    WARN=$((WARN + 1))
}

check_skip() {
    echo -e "  ${CYAN}○${NC} $1 (skipped)"
    SKIP=$((SKIP + 1))
}

# ── Phase 1: Environment ────────────────────────────────────────────────────

header "Phase 1: Environment Validation"

# Go version
if command -v go &>/dev/null; then
    GO_VERSION=$(go version | grep -oP 'go\d+\.\d+(\.\d+)?')
    GO_MINOR=$(echo "$GO_VERSION" | grep -oP '\d+\.\d+' | head -1)
    check_pass "Go installed: $GO_VERSION"

    # Check minimum version (1.23+ for compatibility, 1.25 target)
    if awk "BEGIN{exit !($GO_MINOR >= 1.23)}"; then
        check_pass "Go version >= 1.23"
    else
        check_fail "Go version >= 1.23 required" "Found $GO_VERSION"
    fi
else
    check_fail "Go not installed" "Install Go 1.25+ from https://go.dev/dl/"
    echo -e "\n${RED}Cannot continue without Go. Aborting.${NC}"
    exit 1
fi

# kubectl
if command -v kubectl &>/dev/null; then
    KUBECTL_VERSION=$(kubectl version --client --short 2>/dev/null || kubectl version --client 2>/dev/null | head -1)
    check_pass "kubectl found: $KUBECTL_VERSION"
else
    check_warn "kubectl not found — passthrough commands will fail at runtime"
fi

# golangci-lint
if command -v golangci-lint &>/dev/null; then
    LINT_VERSION=$(golangci-lint --version 2>/dev/null | head -1)
    check_pass "golangci-lint found: $LINT_VERSION"
    HAS_LINT=true
else
    check_warn "golangci-lint not found — lint checks will be skipped"
    HAS_LINT=false
fi

# goreleaser
if command -v goreleaser &>/dev/null; then
    check_pass "goreleaser found"
    HAS_GORELEASER=true
else
    check_warn "goreleaser not found — release validation will be skipped"
    HAS_GORELEASER=false
fi

# ── Phase 2: Module & Dependency Validation ─────────────────────────────────

header "Phase 2: Module & Dependency Validation"

# go.mod exists
if [ -f go.mod ]; then
    MODULE=$(grep '^module' go.mod | awk '{print $2}')
    check_pass "go.mod found: module $MODULE"
else
    check_fail "go.mod not found" "Run 'go mod init github.com/kubilitics/kcli'"
    exit 1
fi

# go.sum exists
if [ -f go.sum ]; then
    check_pass "go.sum present"
else
    check_warn "go.sum not found — run 'go mod tidy'"
fi

# go mod tidy check
echo -e "\n  Running go mod tidy..."
cp go.mod go.mod.backup
cp go.sum go.sum.backup 2>/dev/null || true

if go mod tidy 2>&1 | tee -a "$LOG_FILE"; then
    if diff -q go.mod go.mod.backup &>/dev/null && diff -q go.sum go.sum.backup &>/dev/null 2>/dev/null; then
        check_pass "go mod tidy — no changes needed"
    else
        check_warn "go mod tidy made changes — dependencies were out of sync"
        echo "  → Review changes with: diff go.mod go.mod.backup"
    fi
else
    check_fail "go mod tidy failed" "Check dependency versions in go.mod"
fi
rm -f go.mod.backup go.sum.backup

# Verify no dependency on kcli subfolder
if grep -q '"github.com/kubilitics/kcli/kcli/' go.mod 2>/dev/null; then
    check_fail "go.mod references kcli subfolder" "Remove kcli/ subfolder dependencies"
else
    check_pass "No stale kcli subfolder references in go.mod"
fi

# Check key dependencies present
echo -e "\n  Checking required dependencies..."
REQUIRED_DEPS=(
    "github.com/spf13/cobra"
    "k8s.io/client-go"
    "k8s.io/apimachinery"
    "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

for dep in "${REQUIRED_DEPS[@]}"; do
    if grep -q "$dep" go.mod; then
        check_pass "Dependency: $dep"
    else
        check_fail "Missing dependency: $dep" "Run: go get $dep"
    fi
done

# go mod verify
echo -e "\n  Running go mod verify..."
if go mod verify 2>&1 | tee -a "$LOG_FILE"; then
    check_pass "go mod verify — all checksums match"
else
    check_fail "go mod verify failed" "Run 'go mod download' to re-fetch"
fi

# ── Phase 3: Compilation ────────────────────────────────────────────────────

header "Phase 3: Compilation"

# Build the binary
echo -e "  Building kcli binary..."
BUILD_START=$(date +%s%N)

if go build -o /tmp/kcli-test-binary ./cmd/kcli/ 2>&1 | tee -a "$LOG_FILE"; then
    BUILD_END=$(date +%s%N)
    BUILD_MS=$(( (BUILD_END - BUILD_START) / 1000000 ))
    BINARY_SIZE=$(stat -c%s /tmp/kcli-test-binary 2>/dev/null || stat -f%z /tmp/kcli-test-binary 2>/dev/null)
    BINARY_SIZE_MB=$(echo "scale=1; $BINARY_SIZE / 1048576" | bc)
    check_pass "Binary compiles successfully (${BUILD_MS}ms)"
    check_pass "Binary size: ${BINARY_SIZE_MB}MB"

    # PRD target: < 20MB
    if (( BINARY_SIZE < 20971520 )); then
        check_pass "Binary size under PRD target (< 20MB)"
    else
        check_fail "Binary exceeds PRD target" "Size: ${BINARY_SIZE_MB}MB, target: < 20MB"
    fi
else
    check_fail "Compilation failed" "See errors above"
    echo -e "\n${RED}Build failed. Remaining checks may be incomplete.${NC}"
fi

# Build all packages (including tests)
echo -e "\n  Building all packages..."
if go build ./... 2>&1 | tee -a "$LOG_FILE"; then
    check_pass "All packages compile"
else
    check_fail "Some packages failed to compile" "See errors above"
fi

# ── Phase 4: Static Analysis ────────────────────────────────────────────────

header "Phase 4: Static Analysis"

# go vet
echo -e "  Running go vet..."
if go vet ./... 2>&1 | tee -a "$LOG_FILE"; then
    check_pass "go vet — no issues"
else
    check_fail "go vet found issues" "See errors above"
fi

# golangci-lint
if [ "$HAS_LINT" = true ]; then
    echo -e "  Running golangci-lint..."
    if golangci-lint run ./... --timeout 5m 2>&1 | tee -a "$LOG_FILE"; then
        check_pass "golangci-lint — no issues"
    else
        check_warn "golangci-lint found issues (non-blocking)"
    fi
else
    check_skip "golangci-lint (not installed)"
fi

# Check for common issues manually
echo -e "\n  Checking for common code issues..."

# Unused imports (basic check)
UNUSED_IMPORTS=$(go build ./... 2>&1 | grep -c 'imported and not used' || true)
if [ "$UNUSED_IMPORTS" -eq 0 ]; then
    check_pass "No unused imports"
else
    check_fail "Found $UNUSED_IMPORTS unused import(s)" "Fix with goimports"
fi

# Check for fmt.Println in non-CLI code (should use structured output)
FMT_PRINTLN_COUNT=$(grep -rn 'fmt\.Println' internal/kubectl/ internal/output/ internal/k8sclient/ --include='*.go' 2>/dev/null | grep -v '_test.go' | wc -l || true)
if [ "$FMT_PRINTLN_COUNT" -gt 5 ]; then
    check_warn "Found $FMT_PRINTLN_COUNT fmt.Println calls in library code — consider using structured output"
fi

# Check for context parameter shadowing (the bug we fixed earlier)
SHADOW_COUNT=$(grep -rn 'func.*context string.*context\.Background' internal/ --include='*.go' 2>/dev/null | wc -l || true)
if [ "$SHADOW_COUNT" -eq 0 ]; then
    check_pass "No context parameter shadowing detected"
else
    check_fail "Found $SHADOW_COUNT context parameter shadowing issue(s)" "Rename parameter to 'ctx'"
fi

# ── Phase 5: Unit Tests ─────────────────────────────────────────────────────

header "Phase 5: Unit Tests"

# Count test files
TEST_COUNT=$(find . -name '*_test.go' -not -path './kcli/*' -not -path './vendor/*' | wc -l)
echo -e "  Found ${BOLD}$TEST_COUNT${NC} test files\n"

# Run tests
echo -e "  Running tests..."
if go test ./... -count=1 -timeout 120s 2>&1 | tee -a "$LOG_FILE"; then
    check_pass "All unit tests pass"
else
    check_fail "Some unit tests failed" "See output above"
fi

# Test coverage
echo -e "\n  Generating coverage report..."
if go test ./... -coverprofile=/tmp/kcli-coverage.out -timeout 120s 2>&1 | tee -a "$LOG_FILE"; then
    COVERAGE=$(go tool cover -func=/tmp/kcli-coverage.out 2>/dev/null | grep total | awk '{print $3}')
    check_pass "Test coverage: $COVERAGE"

    # Target: at least 50% coverage
    COVERAGE_NUM=$(echo "$COVERAGE" | tr -d '%')
    if awk "BEGIN{exit !($COVERAGE_NUM >= 50)}"; then
        check_pass "Coverage meets minimum threshold (>= 50%)"
    else
        check_warn "Coverage below 50% — add more tests"
    fi
else
    check_warn "Coverage generation had issues"
fi

# Race detector (only in --full mode)
if [ "$MODE" = "--full" ]; then
    echo -e "\n  Running tests with race detector..."
    if go test -race ./... -count=1 -timeout 300s 2>&1 | tee -a "$LOG_FILE"; then
        check_pass "No race conditions detected"
    else
        check_fail "Race conditions detected" "See output above"
    fi
else
    check_skip "Race detector (use --full to enable)"
fi

# ── Phase 6: Binary Validation ──────────────────────────────────────────────

header "Phase 6: Binary Validation"

if [ -f /tmp/kcli-test-binary ]; then
    BINARY="/tmp/kcli-test-binary"

    # Version command
    echo -e "  Testing kcli version..."
    if VERSION_OUTPUT=$($BINARY version 2>&1); then
        check_pass "kcli version works: $VERSION_OUTPUT"
    else
        check_fail "kcli version failed" "$VERSION_OUTPUT"
    fi

    # Help output
    echo -e "  Testing kcli --help..."
    if HELP_OUTPUT=$($BINARY --help 2>&1); then
        check_pass "kcli --help works"

        # Verify expected commands appear in help
        EXPECTED_CMDS=("ctx" "ns" "get" "health" "restarts" "events" "metrics" "incident" "logs" "ui" "config" "completion" "find" "show" "count" "age" "status" "where" "who" "diff" "blame" "doctor")
        for cmd in "${EXPECTED_CMDS[@]}"; do
            if echo "$HELP_OUTPUT" | grep -qi "$cmd"; then
                check_pass "Command registered: $cmd"
            else
                check_fail "Command missing from help: $cmd" "Check root.go AddCommand"
            fi
        done
    else
        check_fail "kcli --help failed" "$HELP_OUTPUT"
    fi

    # Startup time (PRD target: < 200ms)
    echo -e "\n  Measuring startup time..."
    START_NS=$(date +%s%N)
    $BINARY version >/dev/null 2>&1
    END_NS=$(date +%s%N)
    STARTUP_MS=$(( (END_NS - START_NS) / 1000000 ))
    if [ "$STARTUP_MS" -lt 200 ]; then
        check_pass "Startup time: ${STARTUP_MS}ms (target: < 200ms)"
    elif [ "$STARTUP_MS" -lt 500 ]; then
        check_warn "Startup time: ${STARTUP_MS}ms (target: < 200ms, acceptable: < 500ms)"
    else
        check_fail "Startup time: ${STARTUP_MS}ms" "Target: < 200ms"
    fi

    # Completion generation
    echo -e "\n  Testing completion generation..."
    for shell in bash zsh fish powershell; do
        if $BINARY completion $shell >/dev/null 2>&1; then
            check_pass "Completion: $shell"
        else
            check_fail "Completion failed: $shell" "Check completion command"
        fi
    done

    # Config path
    echo -e "\n  Testing config commands..."
    if $BINARY config path 2>&1 | grep -q '/'; then
        check_pass "kcli config path works"
    else
        check_warn "kcli config path may not be working"
    fi

    # Memory usage (PRD target: < 30MB idle)
    echo -e "\n  Measuring memory usage..."
    if command -v /usr/bin/time &>/dev/null; then
        MEM_OUTPUT=$(/usr/bin/time -v $BINARY version 2>&1 || true)
        MAX_RSS=$(echo "$MEM_OUTPUT" | grep 'Maximum resident' | awk '{print $NF}')
        if [ -n "$MAX_RSS" ]; then
            MAX_RSS_MB=$((MAX_RSS / 1024))
            if [ "$MAX_RSS_MB" -lt 30 ]; then
                check_pass "Memory usage: ${MAX_RSS_MB}MB (target: < 30MB)"
            else
                check_warn "Memory usage: ${MAX_RSS_MB}MB (target: < 30MB)"
            fi
        fi
    else
        check_skip "Memory measurement (GNU time not available)"
    fi

    rm -f /tmp/kcli-test-binary
else
    check_skip "Binary validation (build failed)"
fi

# ── Phase 7: Release Validation (--full mode) ───────────────────────────────

if [ "$MODE" = "--full" ]; then
    header "Phase 7: Release Validation"

    if [ "$HAS_GORELEASER" = true ] && [ -f .goreleaser.yml ]; then
        echo -e "  Running goreleaser check..."
        if goreleaser check 2>&1 | tee -a "$LOG_FILE"; then
            check_pass "goreleaser config valid"
        else
            check_fail "goreleaser config invalid" "Fix .goreleaser.yml"
        fi

        echo -e "  Running goreleaser build (snapshot)..."
        if goreleaser build --snapshot --clean 2>&1 | tee -a "$LOG_FILE"; then
            check_pass "goreleaser snapshot build succeeds"

            # Check all platform binaries
            for platform in darwin-amd64 darwin-arm64 linux-amd64 linux-arm64 windows-amd64; do
                if find dist/ -name "*$platform*" 2>/dev/null | grep -q .; then
                    check_pass "Binary built for: $platform"
                else
                    check_fail "Missing binary for: $platform" "Check .goreleaser.yml targets"
                fi
            done
        else
            check_fail "goreleaser snapshot build failed" "See errors above"
        fi
    else
        check_skip "goreleaser validation (not available or no config)"
    fi
else
    check_skip "Release validation (use --full to enable)"
fi

# ── Phase 8: Benchmarks (--full mode) ───────────────────────────────────────

if [ "$MODE" = "--full" ]; then
    header "Phase 8: Benchmarks"

    echo -e "  Running benchmarks..."
    if go test -bench=. -benchmem -run='^$' ./... -timeout 300s 2>&1 | tee -a "$LOG_FILE"; then
        check_pass "Benchmarks completed"
    else
        check_warn "Some benchmarks failed"
    fi
else
    check_skip "Benchmarks (use --full to enable)"
fi

# ── Summary ──────────────────────────────────────────────────────────────────

header "Validation Summary"

TOTAL=$((PASS + FAIL + WARN + SKIP))

echo -e "  ${GREEN}Passed:  $PASS${NC}"
echo -e "  ${RED}Failed:  $FAIL${NC}"
echo -e "  ${YELLOW}Warnings: $WARN${NC}"
echo -e "  ${CYAN}Skipped: $SKIP${NC}"
echo -e "  ${BOLD}Total:   $TOTAL${NC}"
echo ""
echo -e "  Full log: $LOG_FILE"
echo ""

if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}${BOLD}BUILD VALIDATION PASSED${NC}"
    exit 0
else
    echo -e "  ${RED}${BOLD}BUILD VALIDATION FAILED — $FAIL issue(s) to fix${NC}"
    exit 1
fi
