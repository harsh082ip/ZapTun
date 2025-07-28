#!/bin/bash

set -e

URL_PREFIX="https://github.com/harsh082ip/ZapTun/releases/download/2.4"
INSTALL_DIR=${INSTALL_DIR:-/usr/local/bin}

case "$(uname -sm)" in
  "Darwin x86_64") FILENAME="zaptun-darwin-amd64" ;;
  "Darwin arm64") FILENAME="zaptun-darwin-arm64" ;;
  "Linux x86_64") FILENAME="zaptun-linux-amd64" ;;
  "Linux i686") FILENAME="zaptun-linux-386" ;;
  "Linux armv7l") FILENAME="zaptun-linux-arm" ;;
  "Linux aarch64") FILENAME="zaptun-linux-arm64" ;;
  *) echo "Unsupported architecture: $(uname -sm)" >&2; exit 1 ;;
esac

echo "Downloading $FILENAME from github releases"
if ! curl -sSLf "$URL_PREFIX/$FILENAME" -o "$INSTALL_DIR/zaptun"; then
  echo "Failed to write to $INSTALL_DIR; try with sudo" >&2
  exit 1
fi

if ! chmod +x "$INSTALL_DIR/zaptun"; then
  echo "Failed to set executable permission on $INSTALL_DIR/zaptun" >&2
  exit 1
fi

echo "zaptun is successfully installed"
