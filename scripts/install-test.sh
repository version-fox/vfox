#!/bin/bash
set -eu

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT

BIN_DIR="$TMP_ROOT/bin"
HOME_DIR="$TMP_ROOT/home"
PREFIX_DIR="$TMP_ROOT/prefix"
WORK_DIR="$TMP_ROOT/work"
mkdir -p "$BIN_DIR" "$HOME_DIR" "$PREFIX_DIR" "$WORK_DIR"

cat > "$BIN_DIR/curl" <<'EOF'
#!/bin/bash
set -eu
printf '%s\n' "$@" >> "$VFOX_TEST_ARGS_FILE"
printf '{}\n'
EOF

cp "$BIN_DIR/curl" "$BIN_DIR/wget"
chmod +x "$BIN_DIR/curl" "$BIN_DIR/wget"

run_installer() {
  local args_file="$1"
  shift
  : > "$args_file"
  (
    cd "$WORK_DIR"
    HOME="$HOME_DIR" \
    PREFIX="$PREFIX_DIR" \
    PATH="$BIN_DIR:$PATH" \
    GITHUB_TOKEN="dummy-token" \
    VFOX_TEST_ARGS_FILE="$args_file" \
    bash "$REPO_ROOT/install.sh" --user "$@" >/dev/null 2>&1
  ) || true
}

fail() {
  echo "install-test.sh: $*" >&2
  exit 1
}

DEFAULT_ARGS="$TMP_ROOT/default.args"
run_installer "$DEFAULT_ARGS"
grep -Fxq "Authorization: Bearer dummy-token" "$DEFAULT_ARGS" || fail "default run missing Authorization header"
grep -Fxq "https://api.github.com/repos/version-fox/vfox/releases/latest" "$DEFAULT_ARGS" || fail "default run missing api.github.com URL"

MIRROR_ARGS="$TMP_ROOT/mirror.args"
run_installer "$MIRROR_ARGS" --vfox-installer-github-base-url=https://gh-proxy.org/https://github.com
grep -Fxq "https://gh-proxy.org/https://api.github.com/repos/version-fox/vfox/releases/latest" "$MIRROR_ARGS" || fail "mirror run did not query mirror API URL"
if grep -qi "authorization" "$MIRROR_ARGS"; then
  fail "mirror run unexpectedly sent Authorization header"
fi
if grep -Fq "dummy-token" "$MIRROR_ARGS"; then
  fail "mirror run leaked GITHUB_TOKEN in arguments"
fi

MIRROR_VERSION_ARGS="$TMP_ROOT/mirror-version.args"
run_installer "$MIRROR_VERSION_ARGS" --version 1.0.0 --vfox-installer-github-base-url=https://gh-proxy.org/https://github.com
grep -Fxq "https://gh-proxy.org/https://api.github.com/repos/version-fox/vfox/releases/tags/v1.0.0" "$MIRROR_VERSION_ARGS" || fail "mirror --version run did not query mirror tag URL"
if grep -qi "authorization" "$MIRROR_VERSION_ARGS"; then
  fail "mirror --version run unexpectedly sent Authorization header"
fi
if grep -Fq "dummy-token" "$MIRROR_VERSION_ARGS"; then
  fail "mirror --version run leaked GITHUB_TOKEN in arguments"
fi
