#!/bin/bash

main() {
  # Parse command-line arguments
  USER_INSTALL=false
  VERSION=""
  while [ $# -gt 0 ]; do
    case "$1" in
      --user)
        USER_INSTALL=true
        shift
        ;;
      --version)
        if [ -z "$2" ]; then
          echo "Error: --version requires a value (e.g. --version 0.5.6 or --version v0.5.6)"
          exit 1
        fi
        VERSION="$2"
        shift 2
        ;;
      --version=*)
        VERSION="${1#--version=}"
        if [ -z "$VERSION" ]; then
          echo "Error: --version requires a value (e.g. --version=0.5.6)"
          exit 1
        fi
        shift
        ;;
      *)
        echo "Unknown argument: $1"
        echo "Usage: $0 [--user] [--version <version>]"
        exit 1
        ;;
    esac
  done

  # Strip a leading 'v' from a user-supplied version (we add it back when needed)
  if [ -n "$VERSION" ]; then
    VERSION="${VERSION#v}"
  fi

  # Detect if running in Termux
  IS_TERMUX=false
  case "${HOME:-}" in
    *com.termux*)
      IS_TERMUX=true
      echo "Detected Termux environment"
      ;;
  esac

  # Set installation directory and sudo command based on environment
  if [ "$USER_INSTALL" = true ]; then
    INSTALL_DIR="${HOME}/.local/bin"
    SUDO_CMD=""
    echo "Installing to user directory: $INSTALL_DIR"
  elif [ "$IS_TERMUX" = true ]; then
    INSTALL_DIR="${PREFIX}/bin"
    SUDO_CMD=""
  else
    INSTALL_DIR="/usr/local/bin"
    # Check if sudo is available
    if command -v sudo &> /dev/null; then
      SUDO_CMD="sudo"
    else
      SUDO_CMD=""
    fi
  fi

  # Check if curl or wget is installed
  if command -v curl &> /dev/null
  then
    DOWNLOAD_CMD="curl -LO"
  elif command -v wget &> /dev/null
  then
    DOWNLOAD_CMD="wget"
  else
    echo "Neither curl nor wget was found. Please install one of them and try again."
    exit 1
  fi

  # Get the version: use the user-specified one if provided, otherwise fetch latest
  if [ -n "$VERSION" ]; then
    if [ -n "${GITHUB_TOKEN}" ]; then
      API_RESPONSE=$(curl --silent --header "Authorization: Bearer ${GITHUB_TOKEN}" "https://api.github.com/repos/version-fox/vfox/releases/tags/v${VERSION}")
    else
      API_RESPONSE=$(curl --silent "https://api.github.com/repos/version-fox/vfox/releases/tags/v${VERSION}")
    fi
  elif [ -n "${GITHUB_TOKEN}" ]; then
    API_RESPONSE=$(curl --silent --header "Authorization: Bearer ${GITHUB_TOKEN}" "https://api.github.com/repos/version-fox/vfox/releases/latest")
  else
    API_RESPONSE=$(curl --silent "https://api.github.com/repos/version-fox/vfox/releases/latest")
  fi

  # Check if the response contains an error message
  if echo "$API_RESPONSE" | grep -q '"message":'; then
    echo "GitHub API Error:"
    echo "$API_RESPONSE"
    exit 1
  fi

  RESOLVED_VERSION=$(echo "$API_RESPONSE" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | cut -c 2-)
  if [ -z "$RESOLVED_VERSION" ]; then
    if [ -n "$VERSION" ]; then
      echo "Failed to find vfox version v${VERSION} on GitHub. Please verify the version exists at https://github.com/version-fox/vfox/releases."
    else
      echo "Failed to get the latest version. Please check your network connection and try again."
    fi
    exit 1
  fi
  VERSION="$RESOLVED_VERSION"
  echo "Installing vfox v$VERSION ..."

  # Check if the OS is supported
  OS_TYPE=$(uname -s | tr '[:upper:]' '[:lower:]')
  if [ "$OS_TYPE" = "darwin" ]; then
    OS_TYPE="macos"
  fi

  ARCH_TYPE=$(uname -m)

  if [ "$ARCH_TYPE" = "arm64" ]; then
      ARCH_TYPE="aarch64"
  elif [ "$ARCH_TYPE" = "loongarch64" ]; then
      ARCH_TYPE="loong64"
  fi

  FILENAME="vfox_${VERSION}_${OS_TYPE}_${ARCH_TYPE}"
  TAR_FILE="${FILENAME}.tar.gz"

  echo https://github.com/version-fox/vfox/releases/download/v$VERSION/$TAR_FILE
  $DOWNLOAD_CMD https://github.com/version-fox/vfox/releases/download/v$VERSION/$TAR_FILE


  tar -zxvf $TAR_FILE
  if [ $? -ne 0 ]; then
    echo "Failed to extract vfox binary. Please check if the downloaded file is a valid tar.gz file."
    exit 1
  fi

  # Create installation directory
  $SUDO_CMD mkdir -p "$INSTALL_DIR"
  if [ $? -ne 0 ]; then
    echo "Failed to create $INSTALL_DIR directory. Please check your permissions and try again."
    exit 1
  fi

  if [ -d "$INSTALL_DIR" ]; then
    $SUDO_CMD mv "${FILENAME}/vfox" "$INSTALL_DIR"
  else
    echo "$INSTALL_DIR is not a directory. Please make sure it is a valid directory path."
    exit 1
  fi

  if [ $? -ne 0 ]; then
    echo "Failed to move vfox to $INSTALL_DIR. Please check your permissions and try again."
    exit 1
  fi
  rm $TAR_FILE
  rm -rf $FILENAME
  echo "vfox installed successfully!"

  # Check and update PATH if installing to user directory
  if [ "$USER_INSTALL" = true ]; then
    # Check if ~/.local/bin is in PATH using POSIX-compliant pattern matching
    case ":$PATH:" in
      *":$INSTALL_DIR:"*)
        echo "$INSTALL_DIR is already in your PATH."
        ;;
      *)
        echo ""
        echo "WARNING: $INSTALL_DIR is not in your PATH."
        echo "To add it to your PATH, run one of the following commands based on your shell:"
        echo ""

        # Common export command for bash/zsh
        PATH_EXPORT_CMD='export PATH="$HOME/.local/bin:$PATH"'

        # Detect the current shell more reliably than using $SHELL alone
        if command -v ps >/dev/null 2>&1; then
          CURRENT_SHELL=$(ps -p "$$" -o comm= 2>/dev/null | tr -d ' ')
        fi
        if [ -z "$CURRENT_SHELL" ] && [ -n "$SHELL" ]; then
          CURRENT_SHELL=$(basename "$SHELL")
        fi

        case "$CURRENT_SHELL" in
          bash)
            echo "  For bash:"
            echo "    echo '$PATH_EXPORT_CMD' >> ~/.bashrc"
            echo "    source ~/.bashrc"
            ;;
          zsh)
            echo "  For zsh:"
            echo "    echo '$PATH_EXPORT_CMD' >> ~/.zshrc"
            echo "    source ~/.zshrc"
            ;;
          fish)
            echo "  For fish:"
            echo "    # Run this in a fish shell:"
            echo "    fish_add_path $INSTALL_DIR"
            echo "    # Or persist it by adding this line to your config:"
            echo "    echo 'fish_add_path $INSTALL_DIR' >> ~/.config/fish/config.fish"
            ;;
          *)
            echo "  For bash/zsh:"
            echo "    echo '$PATH_EXPORT_CMD' >> ~/.bashrc  # or ~/.zshrc"
            echo "    source ~/.bashrc  # or source ~/.zshrc"
            ;;
        esac
        echo ""
        echo "Or manually add $INSTALL_DIR to your PATH in your shell's configuration file."
        ;;
    esac
  fi
}

main "$@"
