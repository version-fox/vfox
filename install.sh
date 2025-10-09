#!/bin/bash

main() {
  # Detect if running in Termux
  IS_TERMUX=false
  case "${HOME:-}" in
    *com.termux*)
      IS_TERMUX=true
      echo "Detected Termux environment"
      ;;
  esac

  # Set installation directory and sudo command based on environment
  if [ "$IS_TERMUX" = true ]; then
    INSTALL_DIR="${PREFIX}/bin"
    SUDO_CMD=""
  else
    INSTALL_DIR="/usr/local/bin"
    SUDO_CMD="sudo"
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

  # Get the latest version
  if [ -n "${GITHUB_TOKEN}" ]; then
    API_RESPONSE=$(curl --silent --header "Authorization: Bearer ${GITHUB_TOKEN}" "https://api.github.com/repos/version-fox/vfox/releases/latest")
  else
    API_RESPONSE=$(curl --silent "https://api.github.com/repos/version-fox/vfox/releases/latest")
  fi

  # Check if the response contains an error message
  ERROR_MSG=$(echo "$API_RESPONSE" | grep '"message":' | sed -E 's/.*"message": "([^"]+)".*/\1/')
  if [ -n "$ERROR_MSG" ]; then
    echo "GitHub API Error: $ERROR_MSG"
    DOC_URL=$(echo "$API_RESPONSE" | grep '"documentation_url":' | sed -E 's/.*"documentation_url": "([^"]+)".*/\1/')
    if [ -n "$DOC_URL" ]; then
      echo "Documentation: $DOC_URL"
    fi
    exit 1
  fi

  VERSION=$(echo "$API_RESPONSE" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | cut -c 2-)
  if [ -z "$VERSION" ]; then
    echo "Failed to get the latest version. Please check your network connection and try again."
    exit 1
  fi
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
}

main "$@"
