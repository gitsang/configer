#!/bin/bash
set -e

EXAMPLE_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$EXAMPLE_DIR"

echo "=========================================="
echo "Testing configer map support"
echo "=========================================="

run_test() {
    local test_name="$1"
    shift
    echo ""
    echo "------------------------------------------"
    echo "TEST: $test_name"
    echo "CMD: $@"
    echo "------------------------------------------"
    if "$@"; then
        echo "✓ PASS: $test_name"
    else
        echo "✗ FAIL: $test_name"
        return 1
    fi
}

check_output() {
    local test_name="$1"
    local expected="$2"
    local actual="$3"
    if echo "$actual" | grep -q "$expected"; then
        echo "✓ Output contains: $expected"
    else
        echo "✗ Output missing: $expected"
        echo "Actual output:"
        echo "$actual"
        return 1
    fi
}

echo ""
echo "=========================================="
echo "1. Config file tests"
echo "=========================================="

# Test 1: Full config with all map types
run_test "Full config" go run . -c config.full.yaml

# Test 2: Empty map values
OUTPUT=$(go run . -c config.empty.yaml 2>&1)
run_test "Empty maps" bash -c "go run . -c config.empty.yaml"
check_output "Empty maps output" "labels: {}" "$OUTPUT"

# Test 3: Minimal config (no map fields specified)
OUTPUT=$(go run . -c config.minimal.yaml 2>&1)
run_test "Minimal config" bash -c "go run . -c config.minimal.yaml"

# Test 4: No config file at all
OUTPUT=$(go run . 2>&1 || true)
run_test "No config file" bash -c "go run . || true"

echo ""
echo "=========================================="
echo "2. Environment variable override tests"
echo "=========================================="

# Test 5: map[string]string via JSON
OUTPUT=$(EXAMPLE_SERVER_LABELS='{"env":"staging","region":"eu-west-1"}' go run . -c config.full.yaml 2>&1)
run_test "map[string]string via JSON" bash -c "EXAMPLE_SERVER_LABELS='{\"env\":\"staging\",\"region\":\"eu-west-1\"}' go run . -c config.full.yaml"
check_output "JSON env var" "staging" "$OUTPUT"

# Test 6: map[string]string via individual keys
OUTPUT=$(EXAMPLE_SERVER_LABELS_ENV=staging EXAMPLE_SERVER_LABELS_REGION=eu-west-1 go run . -c config.full.yaml 2>&1)
run_test "map[string]string via individual keys" bash -c "EXAMPLE_SERVER_LABELS_ENV=staging EXAMPLE_SERVER_LABELS_REGION=eu-west-1 go run . -c config.full.yaml"
check_output "Individual keys" "staging" "$OUTPUT"

# Test 7: map[string]int via JSON
OUTPUT=$(EXAMPLE_SERVER_PORTS='{"http":3000,"https":3443}' go run . -c config.full.yaml 2>&1)
run_test "map[string]int via JSON" bash -c "EXAMPLE_SERVER_PORTS='{\"http\":3000,\"https\":3443}' go run . -c config.full.yaml"
check_output "Int JSON" "3000" "$OUTPUT"

# Test 8: map[string]bool via JSON
OUTPUT=$(EXAMPLE_SERVER_FEATURES='{"auth":false,"cache":true}' go run . -c config.full.yaml 2>&1)
run_test "map[string]bool via JSON" bash -c "EXAMPLE_SERVER_FEATURES='{\"auth\":false,\"cache\":true}' go run . -c config.full.yaml"
check_output "Bool JSON" "cache: true" "$OUTPUT"

# Test 9: map[string]Struct via individual keys
OUTPUT=$(EXAMPLE_SERVER_SERVICES_USER_HOST=local.example.com EXAMPLE_SERVER_SERVICES_USER_PORT=9001 go run . -c config.full.yaml 2>&1)
run_test "map[string]Struct via individual keys" bash -c "EXAMPLE_SERVER_SERVICES_USER_HOST=local.example.com EXAMPLE_SERVER_SERVICES_USER_PORT=9001 go run . -c config.full.yaml"
check_output "Struct individual" "local.example.com" "$OUTPUT"

# Test 10: Nested struct map via JSON
OUTPUT=$(EXAMPLE_SERVER_ADVANCED_TAGS='{"team":"frontend","project":"web"}' go run . -c config.full.yaml 2>&1)
run_test "Nested map via JSON" bash -c "EXAMPLE_SERVER_ADVANCED_TAGS='{\"team\":\"frontend\",\"project\":\"web\"}' go run . -c config.full.yaml"
check_output "Nested JSON" "frontend" "$OUTPUT"

echo ""
echo "=========================================="
echo "3. Override behavior tests"
echo "=========================================="

# Test 11: Env var overrides config file value
OUTPUT=$(EXAMPLE_LOG_LEVEL=debug go run . -c config.full.yaml 2>&1)
run_test "Env overrides config" bash -c "EXAMPLE_LOG_LEVEL=debug go run . -c config.full.yaml"
check_output "Env override" "level: debug" "$OUTPUT"

# Test 12: JSON env var replaces entire map from config
OUTPUT=$(EXAMPLE_SERVER_LABELS='{"new_key":"new_value"}' go run . -c config.full.yaml 2>&1)
run_test "JSON env replaces map" bash -c "EXAMPLE_SERVER_LABELS='{\"new_key\":\"new_value\"}' go run . -c config.full.yaml"
check_output "Replace map" "new_key: new_value" "$OUTPUT"

echo ""
echo "=========================================="
echo "4. Edge case tests"
echo "=========================================="

# Test 13: Empty JSON object
OUTPUT=$(EXAMPLE_SERVER_LABELS='{}' go run . -c config.full.yaml 2>&1)
run_test "Empty JSON object" bash -c "EXAMPLE_SERVER_LABELS='{}' go run . -c config.full.yaml"
check_output "Empty JSON" "labels: {}" "$OUTPUT"

# Test 14: Invalid JSON (should be ignored)
OUTPUT=$(EXAMPLE_SERVER_LABELS='not-json' go run . -c config.minimal.yaml 2>&1)
run_test "Invalid JSON ignored" bash -c "EXAMPLE_SERVER_LABELS='not-json' go run . -c config.minimal.yaml"

# Test 15: Special characters in values
OUTPUT=$(EXAMPLE_SERVER_LABELS='{"key":"value with spaces"}' go run . -c config.minimal.yaml 2>&1)
run_test "Special characters" bash -c "EXAMPLE_SERVER_LABELS='{\"key\":\"value with spaces\"}' go run . -c config.minimal.yaml"
check_output "Special chars" "value with spaces" "$OUTPUT"

echo ""
echo "=========================================="
echo "All tests completed!"
echo "=========================================="
