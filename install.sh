#
#    Copyright 2023 [lihan aooohan@gmail.com]
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
#

#!/bin/bash

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
VERSION=$(curl --silent "https://api.github.com/repos/version-fox/vfox/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Failed to get the latest version. Please check your network connection and try again."
  exit 1
fi
echo "Installing vfox $VERSION ..."

# Check if the OS is supported
OS_TYPE=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH_TYPE=$(uname -m)
if [ "$ARCH_TYPE" = "x86_64" ]; then
  ARCH_TYPE="amd64"
elif [ "$ARCH_TYPE" = "aarch64" ]; then
  ARCH_TYPE="arm64"
else
  echo "Unsupported architecture type: $ARCH_TYPE"
  exit 1
fi

FILENAME="version-fox_${VERSION}_${OS_TYPE}_${ARCH_TYPE}.tar.gz"

$DOWNLOAD_CMD https://github.com/version-fox/vfox/releases/download/$VERSION/$FILENAME

tar -zxvf $FILENAME
if [ $? -ne 0 ]; then
  echo "Failed to extract vfox binary. Please check if the downloaded file is a valid tar.gz file."
  exit 1
fi

sudo mv vf /usr/local/bin

if [ $? -ne 0 ]; then
  echo "Failed to move vfox to /usr/local/bin. Please check your sudo permissions and try again."
  exit 1
fi
rm $FILENAME
echo "vfox installed successfully!"