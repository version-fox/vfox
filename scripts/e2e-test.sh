#!/bin/bash

# E2E Test Script for vfox
# Tests vfox functionality directly in current shell

set -e

echo "=========================================="
echo "vfox E2E Test - Direct Shell Integration"
echo "=========================================="

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

VFOX_HOME_DIR="${VFOX_HOME:-$HOME/.vfox}"
USER_VFOX_DIR="$HOME/.vfox"
if [ -d "$HOME/.version-fox" ]; then
    USER_VFOX_DIR="$HOME/.version-fox"
fi

TEST_COUNT=0
PASSED=0
FAILED=0

cleanup() {
    echo ""
    echo "=========================================="
    echo "Cleanup"
    echo "=========================================="
    rm -rf "$VFOX_HOME_DIR/cache/nodejs" || true
    rm -f "$VFOX_HOME_DIR/plugin/nodejs*" || true
    rm -rf "$VFOX_HOME_DIR/sdks" || true
    rm -rf "$VFOX_HOME_DIR/tmp"/* || true
    rm -rf "$USER_VFOX_DIR/tmp"/* || true
    rm -rf "$USER_VFOX_DIR/sdks" || true
    rm -f "$USER_VFOX_DIR/config.yaml" || true
    echo "Cleanup completed"
}

trap cleanup EXIT

run_test() {
    local test_name="$1"
    local test_script="$2"
    local expected_output="$3"

    TEST_COUNT=$((TEST_COUNT + 1))
    echo ""
    echo -e "${YELLOW}Test ${TEST_COUNT}: ${test_name}${NC}"
    echo "Running test..."

    local result
    result=$(eval "${test_script}" 2>&1) || true

    if echo "${result}" | grep -q "${expected_output}"; then
        echo -e "${GREEN}✓ PASSED${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}✗ FAILED${NC}"
        echo "Expected: ${expected_output}"
        echo "Got: ${result}"
        FAILED=$((FAILED + 1))
    fi
}

echo ""
echo "=========================================="
echo "Building vfox"
echo "=========================================="
go build -o vfox .
echo -e "${GREEN}Build completed${NC}"

echo ""
echo "=========================================="
echo "Setup: Activating vfox in current shell"
echo "=========================================="
export PATH="$(pwd):${PATH}"
if ! VFOX_ACTIVATION="$(./vfox activate bash)" 2>&1; then
    echo "Failed to get vfox activation script"
    exit 1
fi
eval "$VFOX_ACTIVATION"
echo -e "${GREEN}vfox activated${NC}"

# Ensure necessary directories exist
mkdir -p "$VFOX_HOME_DIR" || true
mkdir -p "$VFOX_HOME_DIR/plugin" || true
mkdir -p "$VFOX_HOME_DIR/cache" || true
mkdir -p "$VFOX_HOME_DIR/tmp" || true

echo ""
echo "=========================================="
echo "Setup: Installing test SDK"
echo "=========================================="
if ! ./vfox add nodejs 2>&1; then
    echo -e "${RED}Failed to add nodejs plugin${NC}"
    # Don't exit, plugin might already exist
fi
if ! ./vfox install -y nodejs@18.19.0 2>&1; then
    echo -e "${RED}Failed to install nodejs@18.19.0${NC}"
    exit 1
fi
echo -e "${GREEN}Setup completed${NC}"

echo ""
echo "=========================================="
echo "Running E2E Tests"
echo "=========================================="

run_test "Check vfox command available" \
    "which vfox" \
    "vfox"

run_test "Use nodejs in session scope" \
    "vfox use -s nodejs@18.19.0 && vfox current nodejs" \
    "18.19"

run_test "Check current version" \
    "vfox current nodejs" \
    "18.19"

run_test "List available versions" \
    "vfox available | grep nodejs" \
    "nodejs"

run_test "Use nodejs in global scope" \
    "vfox use -g nodejs@18.19.0 && eval \"\$(./vfox activate bash)\" && vfox current nodejs" \
    "18.19"

run_test "Verify global symlink exists" \
    "[ -L ${USER_VFOX_DIR}/sdks/nodejs ] && echo 'SYMLINK_EXISTS'" \
    "SYMLINK_EXISTS"

run_test "Verify global PATH contains vfox nodejs" \
    "eval \"\$(./vfox activate bash)\" && echo \"\$PATH\" | grep -q 'vfox/sdks/nodejs' && echo 'PATH_CORRECT'" \
    "PATH_CORRECT"

run_test "Verify session tmp directory created" \
    "[ -d ${USER_VFOX_DIR}/tmp ] && ls -d ${USER_VFOX_DIR}/tmp/[0-9]* 2>/dev/null | grep -q '[0-9]' && echo 'SESSION_TMP_EXISTS'" \
    "SESSION_TMP_EXISTS"

run_test "Project scope creates .vfox.toml" \
    "(cd /tmp && rm -rf vfox-test-proj && mkdir vfox-test-proj && cd vfox-test-proj && vfox use -p nodejs@18.19.0 && [ -f .vfox.toml ] && echo 'TOML_CREATED')" \
    "TOML_CREATED"

run_test "Verify .vfox.toml contains correct version" \
    "[ -f /tmp/vfox-test-proj/.vfox.toml ] && grep -q '18.19' /tmp/vfox-test-proj/.vfox.toml && echo 'VERSION_CORRECT'" \
    "VERSION_CORRECT"

run_test "Verify project symlink exists" \
    "[ -d /tmp/vfox-test-proj/.vfox/sdks/nodejs ] && echo 'PROJECT_SYMLINK_EXISTS'" \
    "PROJECT_SYMLINK_EXISTS"

run_test "Verify project .vfox directory created" \
    "[ -d /tmp/vfox-test-proj/.vfox ] && [ -d /tmp/vfox-test-proj/.vfox/sdks ] && echo 'PROJECT_VFOX_DIR_OK'" \
    "PROJECT_VFOX_DIR_OK"

run_test "Session scope does not create .vfox.toml" \
    "(cd /tmp && rm -rf vfox-session-test && mkdir vfox-session-test && cd vfox-session-test && vfox use -s nodejs@18.19.0 && [ ! -f .vfox.toml ] && echo 'NO_TOML_CREATED')" \
    "NO_TOML_CREATED"

run_test "Verify global config file location" \
    "[ -f ${USER_VFOX_DIR}/config.yaml ] && echo 'CONFIG_EXISTS'" \
    "CONFIG_EXISTS"

run_test "Verify global SDK installation path" \
    "[ -d ${VFOX_HOME_DIR}/cache/nodejs ] && ls -d ${VFOX_HOME_DIR}/cache/nodejs/v-* 2>/dev/null | head -1 | xargs -I {} test -d {} && echo 'INSTALL_PATH_OK'" \
    "INSTALL_PATH_OK"

run_test "Verify symlink points to correct installation" \
    "[ -L ${USER_VFOX_DIR}/sdks/nodejs ] && readlink ${USER_VFOX_DIR}/sdks/nodejs | grep -q 'cache/nodejs' && echo 'SYMLINK_CORRECT'" \
    "SYMLINK_CORRECT"

run_test "Multiple versions can be installed" \
    "vfox install nodejs@20.11.0 && vfox list nodejs | grep -q '20.11' && vfox list nodejs | grep -q '18.19' && echo 'MULTI_VERSION_OK'" \
    "MULTI_VERSION_OK"

run_test "Use different version in same session" \
    "vfox use -s nodejs@20.11.0 && vfox current nodejs | grep -q '20.11' && echo 'VERSION_SWITCH_OK'" \
    "VERSION_SWITCH_OK"

run_test "Uninstall removes version" \
    "vfox uninstall nodejs@20.11.0 && ! vfox list nodejs | grep -q '20.11' && echo 'UNINSTALL_OK'" \
    "UNINSTALL_OK"

run_test "Verify .vfox.toml format is correct" \
    "[ -f /tmp/vfox-test-proj/.vfox.toml ] && grep -q '^\\[tools\\]' /tmp/vfox-test-proj/.vfox.toml && grep -q '^nodejs' /tmp/vfox-test-proj/.vfox.toml && echo 'FORMAT_OK'" \
    "FORMAT_OK"

run_test "Global scope does not create .vfox.toml in pwd" \
    "(cd /tmp && rm -rf vfox-global-test && mkdir vfox-global-test && cd vfox-global-test && vfox use -g nodejs@18.19.0 && [ ! -f .vfox.toml ] && echo 'NO_LOCAL_TOML')" \
    "NO_LOCAL_TOML"

run_test "Unuse global removes symlink" \
    "vfox unuse -g nodejs && [ ! -L ${USER_VFOX_DIR}/sdks/nodejs ] && echo 'UNLINK_OK'" \
    "UNLINK_OK"

echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo "Total tests: ${TEST_COUNT}"
echo -e "${GREEN}Passed: ${PASSED}${NC}"
echo -e "${RED}Failed: ${FAILED}${NC}"

if [ ${FAILED} -eq 0 ]; then
    echo ""
    echo -e "${GREEN}All tests passed! ✓${NC}"
    exit 0
else
    echo ""
    echo -e "${RED}Some tests failed! ✗${NC}"
    exit 1
fi

