#!/usr/bin/env sh
# agr installer — downloads the latest release binary for your platform.
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/rohithilluri/agent-registry/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/rohithilluri/agent-registry/main/install.sh | sh -s -- --dir /usr/local/bin

set -e

REPO="rohithilluri/agent-registry"
BINARY="agr"
INSTALL_DIR="${AGR_INSTALL_DIR:-/usr/local/bin}"

# Allow --dir override
for arg in "$@"; do
  case "$arg" in
    --dir=*) INSTALL_DIR="${arg#--dir=}" ;;
    --dir)   shift; INSTALL_DIR="$1" ;;
  esac
done

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac
case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS (use the .zip release on Windows)" >&2; exit 1 ;;
esac

# Fetch latest version tag
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
if [ -z "$LATEST" ]; then
  echo "Could not determine latest version" >&2
  exit 1
fi
VERSION="${LATEST#v}"

TARBALL="agr_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${TARBALL}"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${LATEST}/agr_${VERSION}_checksums.txt"

echo "Installing agr ${LATEST} (${OS}/${ARCH}) → ${INSTALL_DIR}"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Download tarball + checksums
curl -fsSL "$URL" -o "${TMP}/${TARBALL}"
curl -fsSL "$CHECKSUMS_URL" -o "${TMP}/checksums.txt"

# Verify checksum
cd "$TMP"
grep "${TARBALL}" checksums.txt | sha256sum -c - || {
  echo "Checksum verification failed!" >&2
  exit 1
}
echo "✓ Checksum verified"

# Extract binary
tar -xzf "${TARBALL}" "${BINARY}"

# Install
mkdir -p "${INSTALL_DIR}"
mv "${BINARY}" "${INSTALL_DIR}/${BINARY}"
chmod +x "${INSTALL_DIR}/${BINARY}"

echo "✓ agr installed to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Run 'agr --help' to get started."
echo "Search the registry: agr search"
