#!/bin/bash

# E2E tests for vfox path command
# This script tests the path command functionality in a controlled environment

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    case $status in
        "PASS")
            echo -e "${GREEN}[PASS]${NC} $message"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            TESTS_TOTAL=$((TESTS_TOTAL + 1))
            ;;
        "FAIL")
            echo -e "${RED}[FAIL]${NC} $message"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            TESTS_TOTAL=$((TESTS_TOTAL + 1))
            ;;
        "INFO")
            echo -e "${YELLOW}[INFO]${NC} $message"
            ;;
    esac
}

# Function to run a test
run_test() {
    local test_name=$1
    local expected_output=$2
    local expected_exit_code=${3:-0}
    shift 3
    local cmd=("$@")

    echo ""
    print_status "INFO" "Running test: $test_name"
    print_status "INFO" "Command: ${cmd[*]}"

    # Capture both stdout and exit code
    local output exit_code
    set +e
    output=$(cd "$TEST_DIR" && "${cmd[@]}" 2>&1)
    exit_code=$?
    set -e

    # Check exit code
    if [[ $exit_code -ne $expected_exit_code ]]; then
        print_status "FAIL" "$test_name - Expected exit code $expected_exit_code, got $exit_code"
        echo "Output: $output"
        return 0  # Don't exit, continue with other tests
    fi

    # Check output (if expected_output is not empty)
    if [[ -n "$expected_output" ]]; then
        if [[ "$output" == *"$expected_output"* ]]; then
            print_status "PASS" "$test_name"
        else
            print_status "FAIL" "$test_name - Output mismatch"
            echo "Expected: $expected_output"
            echo "Got: $output"
        fi
    else
        print_status "PASS" "$test_name"
    fi
}

# Function to run a JSON test
run_json_test() {
    local test_name=$1
    local expected_found=$2
    local expected_path_pattern=$3
    shift 3
    local cmd=("$@")

    echo ""
    print_status "INFO" "Running JSON test: $test_name"
    print_status "INFO" "Command: ${cmd[*]}"

    # Capture output
    local output exit_code
    set +e
    output=$(cd "$TEST_DIR" && "${cmd[@]}" 2>&1)
    exit_code=$?
    set -e

    # Check exit code
    if [[ $exit_code -ne 0 ]]; then
        print_status "FAIL" "$test_name - Expected exit code 0, got $exit_code"
        echo "Output: $output"
        return 0
    fi

    # Parse JSON output
    if command -v jq >/dev/null 2>&1; then
        local found path
        found=$(echo "$output" | jq -r '.found')
        path=$(echo "$output" | jq -r '.path')
        
        if [[ "$found" == "$expected_found" ]]; then
            if [[ -z "$expected_path_pattern" ]] || [[ "$path" == *"$expected_path_pattern"* ]]; then
                print_status "PASS" "$test_name"
            else
                print_status "FAIL" "$test_name - Path pattern mismatch"
                echo "Expected path to contain: $expected_path_pattern"
                echo "Got path: $path"
            fi
        else
            print_status "FAIL" "$test_name - Found value mismatch"
            echo "Expected found: $expected_found"
            echo "Got found: $found"
        fi
    else
        # Fallback for systems without jq
        if [[ "$output" == *"\"found\":\"$expected_found\""* ]]; then
            print_status "PASS" "$test_name"
        else
            print_status "FAIL" "$test_name - JSON output doesn't match expected pattern"
            echo "Expected to contain: \"found\":\"$expected_found\""
            echo "Got: $output"
        fi
    fi
}

# Main test execution
main() {
    local script_dir
    script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local project_dir="$script_dir/.."
    
    # Build the binary
    print_status "INFO" "Building vfox binary..."
    cd "$project_dir"
    
    if ! go build -o test/vfox . 2>&1; then
        print_status "FAIL" "Failed to build vfox binary"
        exit 1
    fi
    
    if [[ ! -f "test/vfox" ]]; then
        print_status "FAIL" "Binary not found after build"
        exit 1
    fi
    
    # Set up test directory
    export TEST_DIR="$project_dir/test"
    local vfox_bin="$TEST_DIR/vfox"
    
    print_status "INFO" "Starting e2e tests for vfox path command"
    
    # Test 1: Invalid parameters
    run_test "No arguments" "invalid parameter" 1 "$vfox_bin" path
    
    # Test 2: Missing version
    run_test "Missing version" "version is required" 1 "$vfox_bin" path nodejs
    
    # Test 3: Empty argument
    run_test "Empty argument" "invalid parameter" 1 "$vfox_bin" path ""
    
    # Test 4: Non-existent SDK (regular output)
    run_test "Non-existent SDK" "notfound" 0 "$vfox_bin" path nonexistent@1.0.0
    
    # Test 5: Non-existent SDK (JSON output)
    run_json_test "Non-existent SDK JSON" "false" "" "$vfox_bin" path --json nonexistent@1.0.0
    
    # Test 6: Help contains the path command
    run_test "Help contains path command" "Get the absolute path" 0 "$vfox_bin" help path
    
    # Print summary
    echo ""
    echo "=========================================="
    print_status "INFO" "Test Summary:"
    print_status "INFO" "Total tests: $TESTS_TOTAL"
    print_status "INFO" "Passed: $TESTS_PASSED"
    print_status "INFO" "Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        print_status "PASS" "All tests passed!"
        exit 0
    else
        print_status "FAIL" "$TESTS_FAILED test(s) failed"
        exit 1
    fi
}

# Run main function
main "$@"