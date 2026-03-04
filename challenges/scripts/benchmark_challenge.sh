#!/usr/bin/env bash
set -euo pipefail
PASS=0; FAIL=0
pass() { PASS=$((PASS+1)); echo "PASS: $1"; }
fail() { FAIL=$((FAIL+1)); echo "FAIL: $1"; }
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MODULE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
[ -f "$MODULE_DIR/go.mod" ] && pass "go.mod exists" || fail "go.mod missing"
[ -f "$MODULE_DIR/benchmark/runner.go" ] && pass "runner.go exists" || fail "runner.go missing"
[ -f "$MODULE_DIR/benchmark/types.go" ] && pass "types.go exists" || fail "types.go missing"
[ -f "$MODULE_DIR/benchmark/integration.go" ] && pass "integration.go exists" || fail "integration.go missing"
[ -f "$MODULE_DIR/CLAUDE.md" ] && pass "CLAUDE.md exists" || fail "CLAUDE.md missing"
[ -f "$MODULE_DIR/README.md" ] && pass "README.md exists" || fail "README.md missing"
[ -f "$MODULE_DIR/AGENTS.md" ] && pass "AGENTS.md exists" || fail "AGENTS.md missing"
(cd "$MODULE_DIR" && go build ./... 2>&1) && pass "module builds successfully" || fail "module build failed"
echo "Results: $PASS passed, $FAIL failed"
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
