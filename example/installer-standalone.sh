#!/usr/bin/env bash
#
# Install urch to /usr/local/bin.
#
# Usage:
#   bash installer-standalone.sh
#
# Env:
#   URCH_VER  version to install, default 1.9.0

set -euo pipefail

URCH_VER="${URCH_VER:-1.9.0}"

# detect the release asset name of the current cpu architecture
case "$(uname -m)" in
  x86_64 | amd64) URCH_ARCH=amd64 ;;
  aarch64 | arm64) URCH_ARCH=arm64 ;;
  armv7*) URCH_ARCH=armv7 ;;
  armv6*) URCH_ARCH=armv6 ;;
  i386 | i686) URCH_ARCH=386 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

# install dependencies
sudo apt-get install -y libsecret-tools wget

# download urch
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
URCH_PKG="urch_${URCH_VER}_linux_${URCH_ARCH}.tar.gz"
URCH_SUMS="urch_${URCH_VER}_checksums.txt"
URCH_URL="https://github.com/soulteary/ubuntu-remote-control-helper/releases/download/v${URCH_VER}"
wget -O "$TMP_DIR/$URCH_PKG" "${URCH_URL}/${URCH_PKG}"
wget -O "$TMP_DIR/$URCH_SUMS" "${URCH_URL}/${URCH_SUMS}"

# verify the archive against the checksums of the release
if ! grep -E "^[0-9a-f]{64}  ${URCH_PKG}\$" "$TMP_DIR/$URCH_SUMS" > "$TMP_DIR/$URCH_PKG.sha256"; then
  echo "no checksum found for ${URCH_PKG} in ${URCH_SUMS}" >&2
  exit 1
fi
(cd "$TMP_DIR" && sha256sum -c "$URCH_PKG.sha256")
tar zxf "$TMP_DIR/$URCH_PKG" -C "$TMP_DIR" urch

# copy urch to executable directory
sudo install -m 755 "$TMP_DIR/urch" /usr/local/bin/urch

echo "urch ${URCH_VER} (${URCH_ARCH}) has been installed to /usr/local/bin/urch"
