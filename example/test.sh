#!/bin/bash
set -e

EXAMPLE_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$EXAMPLE_DIR"

echo "=========================================="
echo "Testing configer complex data structures"
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

# Test 1: Full config with all data structure types
run_test "Full config" go run . -c config.full.yaml

# Test 2: Empty values
OUTPUT=$(go run . -c config.empty.yaml 2>&1)
run_test "Empty values" bash -c "go run . -c config.empty.yaml"
check_output "Empty slice" "hosts: \[\]" "$OUTPUT" || check_output "Empty slice" "hosts: \[\]" "$OUTPUT"
check_output "Empty map" "labels: {}" "$OUTPUT"

# Test 3: Minimal config
OUTPUT=$(go run . -c config.minimal.yaml 2>&1)
run_test "Minimal config" bash -c "go run . -c config.minimal.yaml"

# Test 4: No config file
OUTPUT=$(go run . 2>&1 || true)
run_test "No config file" bash -c "go run . || true"

echo ""
echo "=========================================="
echo "2. Slice type tests"
echo "=========================================="

# Test 5: []string via JSON
OUTPUT=$(EXAMPLE_SERVER_HOSTS='["host-a.example.com","host-b.example.com"]' go run . -c config.minimal.yaml 2>&1)
run_test "[]string via JSON" bash -c 'EXAMPLE_SERVER_HOSTS='"'"'["host-a.example.com","host-b.example.com"]'"'"' go run . -c config.minimal.yaml'
check_output "String slice" "host-a.example.com" "$OUTPUT"

# Test 6: []int via JSON
OUTPUT=$(EXAMPLE_SERVER_PORTS='[3000,3001,3002]' go run . -c config.minimal.yaml 2>&1)
run_test "[]int via JSON" bash -c 'EXAMPLE_SERVER_PORTS='"'"'[3000,3001,3002]'"'"' go run . -c config.minimal.yaml'
check_output "Int slice" "3000" "$OUTPUT"

# Test 7: []bool via JSON
OUTPUT=$(EXAMPLE_SERVER_ENABLED='[true,false,true]' go run . -c config.minimal.yaml 2>&1)
run_test "[]bool via JSON" bash -c 'EXAMPLE_SERVER_ENABLED='"'"'[true,false,true]'"'"' go run . -c config.minimal.yaml'
check_output "Bool slice" "true" "$OUTPUT"

# Test 8: []Struct via JSON
OUTPUT=$(EXAMPLE_SERVER_ENDPOINTS='[{"host":"ep1.example.com","port":8001},{"host":"ep2.example.com","port":8002}]' go run . -c config.minimal.yaml 2>&1)
run_test "[]Struct via JSON" bash -c 'EXAMPLE_SERVER_ENDPOINTS='"'"'[{"host":"ep1.example.com","port":8001},{"host":"ep2.example.com","port":8002}]'"'"' go run . -c config.minimal.yaml'
check_output "Struct slice" "ep1.example.com" "$OUTPUT"

# Test 9: Empty slice
OUTPUT=$(EXAMPLE_SERVER_HOSTS='[]' go run . -c config.full.yaml 2>&1)
run_test "Empty slice" bash -c 'EXAMPLE_SERVER_HOSTS='"'"'[]'"'"' go run . -c config.full.yaml'
check_output "Empty slice" "hosts: \[\]" "$OUTPUT" || true

echo ""
echo "=========================================="
echo "3. Map type tests"
echo "=========================================="

# Test 10: map[string]string via JSON
OUTPUT=$(EXAMPLE_SERVER_LABELS='{"env":"staging","region":"eu-west-1"}' go run . -c config.full.yaml 2>&1)
run_test "map[string]string via JSON" bash -c 'EXAMPLE_SERVER_LABELS='"'"'{"env":"staging","region":"eu-west-1"}'"'"' go run . -c config.full.yaml'
check_output "String map" "staging" "$OUTPUT"

# Test 11: map[string]string via individual keys
OUTPUT=$(EXAMPLE_SERVER_LABELS_ENV=staging EXAMPLE_SERVER_LABELS_REGION=eu-west-1 go run . -c config.full.yaml 2>&1)
run_test "map[string]string via individual keys" bash -c "EXAMPLE_SERVER_LABELS_ENV=staging EXAMPLE_SERVER_LABELS_REGION=eu-west-1 go run . -c config.full.yaml"
check_output "Individual keys" "staging" "$OUTPUT"

# Test 12: map[string]int via JSON
OUTPUT=$(EXAMPLE_SERVER_PORT_MAP='{"http":3000,"https":3443}' go run . -c config.full.yaml 2>&1)
run_test "map[string]int via JSON" bash -c 'EXAMPLE_SERVER_PORT_MAP='"'"'{"http":3000,"https":3443}'"'"' go run . -c config.full.yaml'
check_output "Int map" "3000" "$OUTPUT"

# Test 13: map[string][]string via JSON
OUTPUT=$(EXAMPLE_SERVER_TAGS_BY_SERVICE='{"user":["admin","dev"],"order":["customer"]}' go run . -c config.full.yaml 2>&1)
run_test "map[string][]string via JSON" bash -c 'EXAMPLE_SERVER_TAGS_BY_SERVICE='"'"'{"user":["admin","dev"],"order":["customer"]}'"'"' go run . -c config.full.yaml'
check_output "String slice map" "admin" "$OUTPUT"

# Test 14: map[string]Struct via individual keys
OUTPUT=$(EXAMPLE_SERVER_SERVICES_USER_HOST=local.example.com EXAMPLE_SERVER_SERVICES_USER_PORT=9001 go run . -c config.full.yaml 2>&1)
run_test "map[string]Struct via individual keys" bash -c "EXAMPLE_SERVER_SERVICES_USER_HOST=local.example.com EXAMPLE_SERVER_SERVICES_USER_PORT=9001 go run . -c config.full.yaml"
check_output "Struct map individual" "local.example.com" "$OUTPUT"

# Test 15: map[string]Struct via JSON
OUTPUT=$(EXAMPLE_SERVER_SERVICES='{"user":{"host":"json-user.example.com","port":8888},"order":{"host":"json-order.example.com","port":9999}}' go run . -c config.full.yaml 2>&1)
run_test "map[string]Struct via JSON" bash -c 'EXAMPLE_SERVER_SERVICES='"'"'{"user":{"host":"json-user.example.com","port":8888},"order":{"host":"json-order.example.com","port":9999}}'"'"' go run . -c config.full.yaml'
check_output "Struct map JSON" "json-user.example.com" "$OUTPUT"

echo ""
echo "=========================================="
echo "4. Nested struct tests"
echo "=========================================="

# Test 16: Nested []string via JSON
OUTPUT=$(EXAMPLE_SERVER_ADVANCED_TAGS='["new-tag1","new-tag2"]' go run . -c config.full.yaml 2>&1)
run_test "Nested []string via JSON" bash -c 'EXAMPLE_SERVER_ADVANCED_TAGS='"'"'["new-tag1","new-tag2"]'"'"' go run . -c config.full.yaml'
check_output "Nested slice" "new-tag1" "$OUTPUT"

# Test 15: Nested map[string]string via JSON
OUTPUT=$(EXAMPLE_SERVER_ADVANCED_CONFIG='{"debug":"true","verbose":"false"}' go run . -c config.full.yaml 2>&1)
run_test "Nested map via JSON" bash -c 'EXAMPLE_SERVER_ADVANCED_CONFIG='"'"'{"debug":"true","verbose":"false"}'"'"' go run . -c config.full.yaml'
check_output "Nested map" "debug" "$OUTPUT"

# Test 16: Nested []Struct via JSON
OUTPUT=$(EXAMPLE_SERVER_ADVANCED_BACKENDS='[{"host":"new-backend.example.com","port":9999}]' go run . -c config.full.yaml 2>&1)
run_test "Nested []Struct via JSON" bash -c 'EXAMPLE_SERVER_ADVANCED_BACKENDS='"'"'[{"host":"new-backend.example.com","port":9999}]'"'"' go run . -c config.full.yaml'
check_output "Nested struct slice" "new-backend.example.com" "$OUTPUT"

echo ""
echo "=========================================="
echo "5. Override behavior tests"
echo "=========================================="

# Test 17: Env var overrides config file value
OUTPUT=$(EXAMPLE_LOG_LEVEL=debug go run . -c config.full.yaml 2>&1)
run_test "Env overrides config" bash -c "EXAMPLE_LOG_LEVEL=debug go run . -c config.full.yaml"
check_output "Env override" "level: debug" "$OUTPUT"

# Test 18: JSON env replaces slice from config
OUTPUT=$(EXAMPLE_SERVER_HOSTS='["new-host.example.com"]' go run . -c config.full.yaml 2>&1)
run_test "JSON env replaces slice" bash -c 'EXAMPLE_SERVER_HOSTS='"'"'["new-host.example.com"]'"'"' go run . -c config.full.yaml'
check_output "Replace slice" "new-host.example.com" "$OUTPUT"

# Test 19: JSON env replaces map from config
OUTPUT=$(EXAMPLE_SERVER_LABELS='{"new_key":"new_value"}' go run . -c config.full.yaml 2>&1)
run_test "JSON env replaces map" bash -c 'EXAMPLE_SERVER_LABELS='"'"'{"new_key":"new_value"}'"'"' go run . -c config.full.yaml'
check_output "Replace map" "new_key: new_value" "$OUTPUT"

echo ""
echo "=========================================="
echo "6. Edge case tests"
echo "=========================================="

# Test 20: Empty JSON array
OUTPUT=$(EXAMPLE_SERVER_HOSTS='[]' go run . -c config.minimal.yaml 2>&1)
run_test "Empty JSON array" bash -c 'EXAMPLE_SERVER_HOSTS='"'"'[]'"'"' go run . -c config.minimal.yaml'

# Test 21: Empty JSON object
OUTPUT=$(EXAMPLE_SERVER_LABELS='{}' go run . -c config.minimal.yaml 2>&1)
run_test "Empty JSON object" bash -c 'EXAMPLE_SERVER_LABELS='"'"'{}'"'"' go run . -c config.minimal.yaml'

# Test 22: Invalid JSON (should be ignored)
OUTPUT=$(EXAMPLE_SERVER_HOSTS='not-json' go run . -c config.minimal.yaml 2>&1)
run_test "Invalid JSON ignored" bash -c "EXAMPLE_SERVER_HOSTS='not-json' go run . -c config.minimal.yaml"

# Test 23: Special characters in values
OUTPUT=$(EXAMPLE_SERVER_HOSTS='["host with spaces.example.com"]' go run . -c config.minimal.yaml 2>&1)
run_test "Special characters" bash -c 'EXAMPLE_SERVER_HOSTS='"'"'["host with spaces.example.com"]'"'"' go run . -c config.minimal.yaml'
check_output "Special chars" "host with spaces" "$OUTPUT"

echo ""
echo "=========================================="
echo "All tests completed!"
echo "=========================================="
