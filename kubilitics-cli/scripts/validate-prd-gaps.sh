#!/usr/bin/env bash
# =============================================================================
# kcli PRD Gap Analysis & Validation Script (v2)
# =============================================================================
# Scans the codebase to verify every PRD requirement has a corresponding
# implementation. Reports IMPLEMENTED / PARTIAL / MISSING for each feature.
#
# Usage:
#   ./scripts/validate-prd-gaps.sh
# =============================================================================

set -uo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

IMPLEMENTED=0
PARTIAL=0
MISSING=0

header() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${BOLD}  $1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
}

check_impl() {
    echo -e "  ${GREEN}✓ IMPLEMENTED${NC} — $1"
    IMPLEMENTED=$((IMPLEMENTED + 1))
}

check_partial() {
    echo -e "  ${YELLOW}◐ PARTIAL${NC}     — $1"
    echo -e "    ${DIM}Gap: $2${NC}"
    PARTIAL=$((PARTIAL + 1))
}

check_missing() {
    echo -e "  ${RED}✗ MISSING${NC}     — $1"
    echo -e "    ${DIM}Fix: $2${NC}"
    MISSING=$((MISSING + 1))
}

file_exists() { [ -f "$1" ]; }
file_has() { grep -q "$2" "$1" 2>/dev/null; }
files_have() { grep -rq "$2" "$1" --include='*.go' 2>/dev/null; }

# ─────────────────────────────────────────────────────────────────────────────
# PRINCIPLE 1: ERROR HANDLING
# ─────────────────────────────────────────────────────────────────────────────

header "Principle 1: It Must Never Fail — Error Handling"

# kubectl not found — runner/kubectl.go has getKubectlBinary with LookPath fallback
if grep -rq "LookPath\|kubectlBinary\|getKubectlBinary\|FindKubectl" internal/runner/ internal/kubectl/ 2>/dev/null; then
    check_impl "kubectl-not-found error handling"
else
    check_missing "kubectl-not-found error message" "Add clear error when kubectl binary isn't found"
fi

# Kubeconfig error handling
if grep -rq "wrapConfigErr\|ConfigErr\|kubeconfig" internal/k8sclient/ internal/errors/ 2>/dev/null; then
    check_impl "Kubeconfig error handling"
else
    check_missing "Kubeconfig error handling" "Add structured errors for kubeconfig issues"
fi

# Cluster unreachable
if files_have "internal/" "connection refused\|cannot connect\|unreachable\|dial.*tcp\|timeout"; then
    check_impl "Cluster unreachable error detection"
else
    check_partial "Cluster unreachable error" "Add user-friendly cluster connection errors"
fi

# Error classification
if file_exists "internal/errors/errors.go"; then
    check_impl "Error classification system (internal/errors/)"
else
    check_missing "Error classification system" "Create internal/errors/ with typed errors and hints"
fi

# ─────────────────────────────────────────────────────────────────────────────
# PRINCIPLE 2: ZERO STARTUP OVERHEAD
# ─────────────────────────────────────────────────────────────────────────────

header "Principle 2: Zero Startup Overhead"

if ! files_have "cmd/kcli/" "update.*check\|checkForUpdate\|auto.?update"; then
    check_impl "No update check on startup"
else
    check_missing "No update check" "Remove any update check from startup path"
fi

if ! files_have "internal/" "telemetry\|analytics\|tracking\|phone.?home"; then
    check_impl "No telemetry code"
else
    check_missing "No telemetry" "Remove all telemetry/analytics code"
fi

# Lazy client-go: k8sclient has caching with TTL
if files_have "internal/k8sclient/" "Cache\|cache\|sync.\|TTL\|ttl\|mu\b"; then
    check_impl "Lazy client-go initialization with caching"
else
    check_partial "Lazy client-go initialization" "Ensure client-go is only initialized on demand"
fi

# ─────────────────────────────────────────────────────────────────────────────
# PRINCIPLE 3: KUBECTL IS THE TRUTH
# ─────────────────────────────────────────────────────────────────────────────

header "Principle 3: kubectl Passthrough"

if grep -rq "exec.Command" internal/runner/ internal/kubectl/ 2>/dev/null; then
    check_impl "kubectl subprocess delegation"
else
    check_missing "kubectl subprocess delegation" "Implement kubectl passthrough"
fi

# Signal forwarding — check main.go and kubectl passthrough
if grep -rq "signal\|SIGTERM\|SIGINT" cmd/kcli/main.go internal/kubectl/passthrough.go 2>/dev/null; then
    check_impl "Signal forwarding to kubectl subprocess"
else
    check_missing "Signal forwarding" "Forward SIGTERM/SIGINT"
fi

# Unknown flags passthrough
if files_have "internal/cli/" "DisableFlagParsing\|FParseErrWhitelist\|ArbitraryArgs\|TraverseChildren\|passThroughArgs\|PassthroughArgs\|unknownFlags\|remainingArgs"; then
    check_impl "Unknown flags passed through to kubectl"
else
    check_partial "Unknown flags passthrough" "Ensure kcli passes unrecognized flags to kubectl"
fi

# ─────────────────────────────────────────────────────────────────────────────
# SECTION 3.1: CONTEXT MANAGEMENT
# ─────────────────────────────────────────────────────────────────────────────

header "§3.1 Context Management (kcli ctx)"

CTX_FILE="internal/cli/context.go"
if file_exists "$CTX_FILE"; then
    check_impl "ctx command file exists"

    file_has "$CTX_FILE" "Interactive\|interactive\|picker\|Pick\|pick\|fuzzy\|Fuzzy\|tea.Program" && \
        check_impl "ctx: Interactive picker" || \
        check_missing "ctx: Interactive picker" "Add interactive context selection"

    file_has "$CTX_FILE" 'previous\|prev.*context\|switch.*-\|args\[0\].*==.*"-"' && \
        check_impl "ctx: Switch to previous (-)" || \
        check_missing "ctx: Switch to previous (-)" "Support 'kcli ctx -'"

    file_has "$CTX_FILE" '"list"\|List\|ListCmd\|ctx.*list\|--list\|-l' && \
        check_impl "ctx: list subcommand" || \
        check_missing "ctx: list subcommand" "Add 'kcli ctx list'"

    file_has "$CTX_FILE" '"rename"\|Rename' && \
        check_impl "ctx: rename subcommand" || \
        check_missing "ctx: rename subcommand" "Add 'kcli ctx rename'"

    file_has "$CTX_FILE" '"delete"\|Delete\|remove\|Remove' && \
        check_impl "ctx: delete subcommand" || \
        check_missing "ctx: delete subcommand" "Add 'kcli ctx delete'"

    file_has "$CTX_FILE" 'info\|Info\|cluster.*endpoint\|CurrentContext\|user.*namespace' && \
        check_impl "ctx: info / current display" || \
        check_missing "ctx: info" "Add 'kcli ctx info'"

    file_has "$CTX_FILE" 'export\|Export\|standalone.*kubeconfig\|WriteToFile' && \
        check_impl "ctx: export subcommand" || \
        check_missing "ctx: export" "Add 'kcli ctx export'"
else
    check_missing "ctx command file" "Create internal/cli/context.go"
fi

# ─────────────────────────────────────────────────────────────────────────────
# SECTION 3.2: NAMESPACE MANAGEMENT
# ─────────────────────────────────────────────────────────────────────────────

header "§3.2 Namespace Management (kcli ns)"

NS_FILE="internal/cli/namespace.go"
if file_exists "$NS_FILE"; then
    check_impl "ns command file exists"

    file_has "$NS_FILE" 'previous\|prev.*ns\|switch.*-\|"-"' && \
        check_impl "ns: Switch to previous (-)" || \
        check_missing "ns: Switch to previous (-)" "Support 'kcli ns -'"

    file_has "$NS_FILE" '"list"\|List\|ListCmd\|--list\|-l' && \
        check_impl "ns: list subcommand" || \
        check_missing "ns: list" "Add 'kcli ns list'"

    file_has "$NS_FILE" '"create"\|Create\|create.*namespace' && \
        check_impl "ns: create subcommand" || \
        check_missing "ns: create" "Add 'kcli ns create'"

    file_has "$NS_FILE" '"delete"\|Delete\|delete.*namespace' && \
        check_impl "ns: delete subcommand" || \
        check_missing "ns: delete" "Add 'kcli ns delete'"
else
    check_missing "ns command file" "Create internal/cli/namespace.go"
fi

# ─────────────────────────────────────────────────────────────────────────────
# SECTION 3.3: ENHANCED OUTPUT — WITH MODIFIERS
# ─────────────────────────────────────────────────────────────────────────────

header "§3.3 Enhanced 'with' Modifier System"

if file_exists "internal/kubectl/enhancer.go"; then
    check_impl "'with' modifier parser (enhancer.go)"
else
    check_missing "'with' modifier parser" "Create internal/kubectl/enhancer.go"
fi

# Check enhancer files
for enhfile in enhance_pods enhance_deploy enhance_svc enhance_nodes enhance_misc; do
    if file_exists "internal/kubectl/${enhfile}.go"; then
        check_impl "Enhancer: ${enhfile}.go"
    else
        check_missing "Enhancer: ${enhfile}.go" "Create internal/kubectl/${enhfile}.go"
    fi
done

# Check specific "with" modifiers from PRD — look in enhance_pods.go
echo -e "\n${CYAN}  Checking 'with' modifiers from PRD:${NC}"
PODS_ENH="internal/kubectl/enhance_pods.go"
if file_exists "$PODS_ENH"; then
    for mod in ip node labels images restarts sc all; do
        if file_has "$PODS_ENH" "\"$mod\""; then
            check_impl "pods with $mod"
        else
            check_missing "pods with $mod" "Add $mod case to enhance_pods.go"
        fi
    done
fi

DEPLOY_ENH="internal/kubectl/enhance_deploy.go"
if file_exists "$DEPLOY_ENH"; then
    for mod in replicas images strategy; do
        if file_has "$DEPLOY_ENH" "\"$mod\""; then
            check_impl "deploy with $mod"
        else
            check_missing "deploy with $mod" "Add $mod case to enhance_deploy.go"
        fi
    done
fi

NODE_ENH="internal/kubectl/enhance_nodes.go"
if file_exists "$NODE_ENH"; then
    for mod in capacity taints version pods; do
        if file_has "$NODE_ENH" "\"$mod\""; then
            check_impl "nodes with $mod"
        else
            check_missing "nodes with $mod" "Add $mod case to enhance_nodes.go"
        fi
    done
fi

SVC_ENH="internal/kubectl/enhance_svc.go"
if file_exists "$SVC_ENH"; then
    for mod in endpoints ports; do
        if grep -q "\"$mod\"" "$SVC_ENH" 2>/dev/null; then
            check_impl "svc with $mod"
        else
            check_missing "svc with $mod" "Add $mod case to enhance_svc.go"
        fi
    done
fi

# ─────────────────────────────────────────────────────────────────────────────
# SECTIONS 3.4-3.8: OBSERVABILITY
# ─────────────────────────────────────────────────────────────────────────────

header "§3.4-3.8 Observability Commands"

# Health — defined in observability.go
if grep -q "newHealthCmd" "internal/cli/observability.go" 2>/dev/null || \
   grep -q "newHealthCmd" "internal/cli/health.go" 2>/dev/null; then
    check_impl "health command"
    grep -rq "watch\|Watch" internal/cli/observability.go 2>/dev/null && check_impl "health --watch" || \
        check_partial "health --watch" "Add watch mode"
    grep -rq "json\|JSON\|output.*format" internal/cli/observability.go 2>/dev/null && check_impl "health --output json" || \
        check_partial "health --output json" "Add JSON output"
else
    check_missing "health command" "Implement kcli health"
fi

# Restarts — defined in observability.go
if grep -q "newRestartsCmd" "internal/cli/observability.go" 2>/dev/null; then
    check_impl "restarts command"
    grep -q "min" internal/cli/observability.go 2>/dev/null && check_impl "restarts --min" || \
        check_missing "restarts --min" "Add --min flag"
    grep -q "since\|Since" internal/cli/observability.go 2>/dev/null && check_impl "restarts --since" || \
        check_missing "restarts --since" "Add --since flag"
else
    check_missing "restarts command" "Implement kcli restarts"
fi

# Events — defined in observability.go
if grep -q "newEventsCmd" "internal/cli/observability.go" 2>/dev/null; then
    check_impl "events command"
    grep -q "type\|Type\|fieldSelector" internal/cli/observability.go 2>/dev/null && check_impl "events --type" || \
        check_missing "events --type" "Add --type flag"
    grep -q "reason\|Reason" internal/cli/observability.go 2>/dev/null && check_impl "events --reason" || \
        check_missing "events --reason" "Add --reason flag"
    grep -q "involvedObject\|--for\|forResource" internal/cli/observability.go 2>/dev/null && check_impl "events --for" || \
        check_partial "events --for" "Add --for resource flag"
else
    check_missing "events command" "Implement kcli events"
fi

# Metrics — defined in observability.go
if grep -q "newMetricsCmd" "internal/cli/observability.go" 2>/dev/null; then
    check_impl "metrics command"
else
    check_missing "metrics command" "Implement kcli metrics"
fi

# Incident
if grep -q "newIncidentCmd" "internal/cli/incident.go" 2>/dev/null; then
    check_impl "incident command"
    grep -q "export\|Export" internal/cli/incident.go 2>/dev/null && check_impl "incident --export" || \
        check_partial "incident --export" "Add --export flag"
else
    check_missing "incident command" "Implement kcli incident"
fi

# ─────────────────────────────────────────────────────────────────────────────
# SECTION 3.9: MULTI-POD LOGS
# ─────────────────────────────────────────────────────────────────────────────

header "§3.9 Multi-Pod Log Tailing"

if file_exists "internal/cli/logs.go"; then
    check_impl "logs command"
    file_has "internal/cli/logs.go" "deploy\|Deployment\|labelSelector\|-l\b" && \
        check_impl "logs: deployment/label selector support" || \
        check_missing "logs: deployment support" "Add deploy/ prefix"
    file_has "internal/cli/logs.go" "follow\|Follow\|-f\b" && \
        check_impl "logs: follow mode" || check_missing "logs: follow" "Add -f flag"
    file_has "internal/cli/logs.go" "color\|Color\|NoColor\|no-color" && \
        check_impl "logs: color per pod / --no-color" || check_partial "logs: color" "Add per-pod coloring"
else
    check_missing "logs command" "Implement multi-pod log tailing"
fi

# ─────────────────────────────────────────────────────────────────────────────
# SECTION 3.10: TUI
# ─────────────────────────────────────────────────────────────────────────────

header "§3.10 TUI Dashboard (kcli ui)"

if [ -d "internal/ui" ]; then
    check_impl "TUI directory (internal/ui/)"
    grep -rq "bubbletea\|tea.Model\|tea.Program" internal/ui/ 2>/dev/null && \
        check_impl "TUI: Bubble Tea framework" || check_missing "TUI: Bubble Tea" "Use bubbletea"
    grep -rq "key.Up\|key.Down\|KeyJ\|KeyK\|keyMap" internal/ui/ 2>/dev/null && \
        check_impl "TUI: Keyboard navigation" || check_partial "TUI: Vim keys" "Add j/k navigation"
    grep -rq "Log\|log\|logView\|LogView" internal/ui/ 2>/dev/null && \
        check_impl "TUI: Log viewer" || check_partial "TUI: Log viewer" "Add log viewer"
    grep -rq "yaml\|Yaml\|YAML\|describeView" internal/ui/ 2>/dev/null && \
        check_impl "TUI: YAML viewer" || check_partial "TUI: YAML viewer" "Add YAML viewer"
    grep -rq "exec\|Exec\|shell\|Shell" internal/ui/ 2>/dev/null && \
        check_impl "TUI: Shell exec" || check_partial "TUI: Shell exec" "Add exec into pod"
    grep -rq "portforward\|PortForward" internal/ui/ 2>/dev/null && \
        check_impl "TUI: Port forwarding" || check_partial "TUI: Port forward" "Add port-forward"
else
    check_missing "TUI directory" "Create internal/ui/ with Bubble Tea"
fi

# ─────────────────────────────────────────────────────────────────────────────
# SECTION 3.11: SAFETY MODEL
# ─────────────────────────────────────────────────────────────────────────────

header "§3.11 Safety Model"

if file_exists "internal/kubectl/safety.go"; then
    check_impl "Safety module (internal/kubectl/safety.go)"
    grep -q "RiskNone\|RiskLow\|RiskCritical" "internal/kubectl/safety.go" 2>/dev/null && \
        check_impl "Risk levels (None/Low/Medium/High/Critical)" || \
        check_missing "Risk levels" "Add 5-tier risk classification"
else
    check_missing "Safety module" "Create internal/kubectl/safety.go"
fi

# --yes flag
if grep -rq "\-\-yes" internal/cli/root.go internal/runner/ 2>/dev/null; then
    check_impl "--yes flag to bypass confirmation"
else
    check_missing "--yes flag" "Add --yes global flag"
fi

# KCLI_CONFIRM env var
if files_have "internal/" "KCLI_CONFIRM"; then
    check_impl "KCLI_CONFIRM env var for scripts"
else
    check_missing "KCLI_CONFIRM env var" 'Add os.Getenv("KCLI_CONFIRM") check'
fi

# ─────────────────────────────────────────────────────────────────────────────
# SECTION 3.12: CONFIGURATION
# ─────────────────────────────────────────────────────────────────────────────

header "§3.12 Configuration (kcli config)"

if file_exists "internal/config/config.go"; then
    check_impl "Config system"
    grep -q "SafetyConfig\|RequireConfirm\|CriticalNamespaces" "internal/config/config.go" 2>/dev/null && \
        check_impl "Config: safety section" || check_missing "Config: safety" "Add safety config"
    grep -q "DefaultOutputFormat\|TableStyle\|OutputConfig" "internal/config/config.go" 2>/dev/null && \
        check_impl "Config: output section" || check_missing "Config: output" "Add output config"
    grep -q "TUIConfig\|RefreshInterval" "internal/config/config.go" 2>/dev/null && \
        check_impl "Config: TUI section" || check_missing "Config: TUI" "Add TUI config"
    grep -q "LogsConfig\|MaxLines" "internal/config/config.go" 2>/dev/null && \
        check_impl "Config: logs section" || check_missing "Config: logs" "Add logs config"
else
    check_missing "Config system" "Create internal/config/config.go"
fi

# Config CLI commands
if files_have "internal/cli/config.go" "view\|get\|set\|path\|edit\|reset"; then
    check_impl "Config subcommands (view/get/set/path/edit/reset)"
else
    check_partial "Config subcommands" "Add all config subcommands"
fi

# ─────────────────────────────────────────────────────────────────────────────
# SECTION 3.13: SHELL INTEGRATION
# ─────────────────────────────────────────────────────────────────────────────

header "§3.13 Shell Integration"

file_exists "internal/cli/completion.go" && check_impl "Shell completion (bash/zsh/fish/powershell)" || \
    check_missing "Shell completion" "Add completion command"

file_exists "internal/cli/prompt.go" && check_impl "Prompt integration" || \
    check_missing "Prompt integration" "Add kcli prompt command"

grep -q "newVersionCmd\|VersionCmd" internal/cli/root.go 2>/dev/null && check_impl "Version command" || \
    check_missing "Version command" "Add kcli version"

# ─────────────────────────────────────────────────────────────────────────────
# SECTION 3.14: NATURAL LANGUAGE ALIASES
# ─────────────────────────────────────────────────────────────────────────────

header "§3.14 Natural Language Aliases"

for cmd_name in show find count status who where age diff; do
    if file_exists "internal/cli/${cmd_name}.go" || file_exists "internal/cli/${cmd_name}Cmd.go"; then
        check_impl "Command: kcli $cmd_name"
    else
        check_missing "Command: kcli $cmd_name" "Create internal/cli/${cmd_name}.go"
    fi
done

# ─────────────────────────────────────────────────────────────────────────────
# SECURITY
# ─────────────────────────────────────────────────────────────────────────────

header "Security Requirements"

if ! files_have "internal/" "telemetry\|analytics\|sendMetric\|tracking"; then
    check_impl "No telemetry code"
else
    check_missing "No telemetry" "Remove telemetry code"
fi

if grep -rq "mask\|Mask\|Redact\|redact\|REDACTED" internal/config/ internal/output/ 2>/dev/null; then
    check_impl "Secret/sensitive value masking"
else
    check_missing "Secret masking" "Add masking for sensitive values"
fi

# ─────────────────────────────────────────────────────────────────────────────
# OUTPUT ENGINE
# ─────────────────────────────────────────────────────────────────────────────

header "Output Engine (internal/output/)"

for f in table.go color.go termcaps.go responsive.go progress.go prompt.go status.go error.go formatters.go; do
    file_exists "internal/output/$f" && check_impl "Output: $f" || \
        check_missing "Output: $f" "Create internal/output/$f"
done

# Theme support
grep -q "darkTheme\|lightTheme\|Theme" "internal/output/color.go" 2>/dev/null && \
    check_impl "Color themes (dark/light/auto)" || \
    check_missing "Color themes" "Add theme detection in color.go"

# Color depth degradation
grep -q "TrueColor\|Color256\|ColorDepth" "internal/output/termcaps.go" 2>/dev/null && \
    check_impl "Color depth degradation (TrueColor/256/16/none)" || \
    check_missing "Color depth degradation" "Add color detection in termcaps.go"

# Responsive tables
grep -q "Breakpoint\|DetectBreakpoint" "internal/output/responsive.go" 2>/dev/null && \
    check_impl "Responsive table layout" || \
    check_missing "Responsive tables" "Add breakpoint-based layout"

# Human-readable durations
grep -q "FormatAge\|FormatDuration" "internal/output/formatters.go" 2>/dev/null && \
    check_impl "Human-readable durations" || \
    check_missing "Human-readable durations" "Add FormatAge/FormatDuration"

# Status color coding
grep -q "StatusColor\|PodPhase\|Running\|CrashLoop" "internal/output/status.go" 2>/dev/null && \
    check_impl "Status color coding" || \
    check_partial "Status color coding" "Add per-status color styling"

# ─────────────────────────────────────────────────────────────────────────────
# COMPATIBILITY & RELEASE
# ─────────────────────────────────────────────────────────────────────────────

header "Compatibility & Release"

GO_VERSION=$(grep '^go ' go.mod | awk '{print $2}')
check_impl "Go version in go.mod: $GO_VERSION"

grep -q 'k8s.io/client-go' go.mod && check_impl "client-go dependency" || \
    check_missing "client-go" "Add k8s.io/client-go to go.mod"

# goreleaser
if [ -f ".goreleaser.yml" ] || [ -f ".goreleaser.yaml" ]; then
    check_impl "goreleaser config"
else
    check_missing "goreleaser config in main repo root" \
        "Create .goreleaser.yml with darwin/linux/windows targets (exists in kcli/ subfolder)"
fi

# ─────────────────────────────────────────────────────────────────────────────
# SUMMARY
# ─────────────────────────────────────────────────────────────────────────────

header "PRD Gap Analysis Summary"

TOTAL=$((IMPLEMENTED + PARTIAL + MISSING))
IMPL_PCT=0
if [ "$TOTAL" -gt 0 ]; then
    IMPL_PCT=$((IMPLEMENTED * 100 / TOTAL))
fi

echo -e "  ${GREEN}Implemented: $IMPLEMENTED${NC}"
echo -e "  ${YELLOW}Partial:     $PARTIAL${NC}"
echo -e "  ${RED}Missing:     $MISSING${NC}"
echo -e "  ${BOLD}Total:       $TOTAL checks${NC}"
echo ""
echo -e "  ${BOLD}Implementation coverage: ${IMPL_PCT}%${NC}"
echo ""

if [ "$MISSING" -eq 0 ] && [ "$PARTIAL" -eq 0 ]; then
    echo -e "  ${GREEN}${BOLD}ALL PRD REQUIREMENTS IMPLEMENTED${NC}"
elif [ "$MISSING" -eq 0 ]; then
    echo -e "  ${YELLOW}${BOLD}All features present, $PARTIAL need refinement${NC}"
elif [ "$MISSING" -le 3 ]; then
    echo -e "  ${YELLOW}${BOLD}Nearly complete — $MISSING items missing, $PARTIAL need refinement${NC}"
else
    echo -e "  ${RED}${BOLD}$MISSING features missing, $PARTIAL need refinement${NC}"
fi

echo ""
echo -e "  ${BOLD}Next steps:${NC}"
echo -e "  1. Fix MISSING items"
echo -e "  2. Refine PARTIAL items"
echo -e "  3. Run: ./scripts/validate-build.sh"
echo -e "  4. Run: ./scripts/validate-commands-e2e.sh"

exit $MISSING
