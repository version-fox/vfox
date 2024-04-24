#!/bin/bash

main() {
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
  VERSION=$(curl --silent "https://api.github.com/repos/version-fox/vfox/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' | cut -c 2-)

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

<<<<<<< HEAD
  if [ "$ARCH_TYPE" = "arm64" ]; then
      ARCH_TYPE="aarch64"
  elif [ "$ARCH_TYPE" = "loongarch64" ]; then
      ARCH_TYPE="loong64"
  fi
=======
if [ "$ARCH_TYPE" = "arm64" ]; then
    ARCH_TYPE="aarch64"
fi

FILENAME="vfox_${VERSION}_${OS_TYPE}_${ARCH_TYPE}"
TAR_FILE="${FILENAME}.tar.gz"
>>>>>>> 830d70646b53e2270e4356977065ddb9a81accaa

  FILENAME="vfox_${VERSION}_${OS_TYPE}_${ARCH_TYPE}"
  TAR_FILE="${FILENAME}.tar.gz"

  echo https://github.com/version-fox/vfox/releases/download/v$VERSION/$TAR_FILE
  $DOWNLOAD_CMD https://github.com/version-fox/vfox/releases/download/v$VERSION/$TAR_FILE


  tar -zxvf $TAR_FILE
  if [ $? -ne 0 ]; then
    echo "Failed to extract vfox binary. Please check if the downloaded file is a valid tar.gz file."
    exit 1
  fi

  sudo mv "${FILENAME}/vfox" /usr/local/bin

  if [ $? -ne 0 ]; then
    echo "Failed to move vfox to /usr/local/bin. Please check your sudo permissions and try again."
    exit 1
  fi
  rm $TAR_FILE
  rm -rf $FILENAME
  echo "vfox installed successfully!"
}

main "$@"