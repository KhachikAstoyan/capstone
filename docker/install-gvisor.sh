#!/usr/bin/env bash
set -euo pipefail

# Install gVisor (runsc) and register as Docker runtime
# Supports: Linux (Debian/Ubuntu/RHEL/Rocky), macOS (via Homebrew)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OS=$(uname -s)

usage() {
  cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Install gVisor (runsc) and register as Docker runtime.

OPTIONS:
  --release VERSION    Use specific gVisor release (default: latest from storage.googleapis.com)
  --install-dir PATH   Installation directory (default: /usr/local/bin)
  --no-register        Install gVisor but don't register with Docker
  -h, --help           Show this help message

EXAMPLES:
  # Install latest gVisor and register with Docker
  $(basename "$0")

  # Install specific version
  $(basename "$0") --release 20250430

  # Install to custom directory
  $(basename "$0") --install-dir /opt/gvisor

REQUIREMENTS:
  - Linux: sudo access required
  - macOS: Homebrew installed (brew)
  - Docker installed and running

EOF
}

# Default values
RELEASE="latest"
INSTALL_DIR="/usr/local/bin"
REGISTER_DOCKER=true

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    --release)
      RELEASE="${2:?missing value for --release}"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="${2:?missing value for --install-dir}"
      shift 2
      ;;
    --no-register)
      REGISTER_DOCKER=false
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

echo "==> Installing gVisor (runsc)"
echo "    OS: $OS"
echo "    Release: $RELEASE"
echo "    Install dir: $INSTALL_DIR"

# macOS installation
if [[ "$OS" == "Darwin" ]]; then
  echo "==> macOS detected. Using Homebrew..."

  if ! command -v brew &> /dev/null; then
    echo "ERROR: Homebrew not found. Install from https://brew.sh" >&2
    exit 1
  fi

  brew install gvisor

  RUNSC=$(which runsc)
  echo "✓ gVisor installed: $RUNSC"
  runsc --version

  echo ""
  echo "==> gVisor is installed. Docker runtime registration not required on macOS."
  echo "    (gVisor runs in a Linux VM managed by Docker Desktop)"
  exit 0
fi

# Linux installation
if [[ "$OS" != "Linux" ]]; then
  echo "ERROR: Unsupported OS: $OS" >&2
  exit 1
fi

echo "==> Linux detected. Installing from release binaries..."

# Check prerequisites
if [[ $EUID -ne 0 ]]; then
  echo "ERROR: This script must be run as root on Linux" >&2
  echo "       Try: sudo $0" >&2
  exit 1
fi

# Verify kernel version
KERNEL_VERSION=$(uname -r | cut -d. -f1-2)
MIN_KERNEL="4.14"
if ! (echo -e "$KERNEL_VERSION\n$MIN_KERNEL" | sort -V | head -n1 | grep -q "^$MIN_KERNEL"); then
  echo "WARNING: Kernel $KERNEL_VERSION detected. gVisor requires 4.14+. Proceeding anyway..." >&2
fi

# Download runsc
if [[ "$RELEASE" == "latest" ]]; then
  RELEASE=$(date +%Y%m%d)
fi

echo "==> Downloading runsc (release: $RELEASE)..."
DOWNLOAD_URL="https://storage.googleapis.com/gvisor/releases/release-${RELEASE}/x86_64/runsc"
CHECKSUM_URL="${DOWNLOAD_URL}.sha512"

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

cd "$TMP_DIR"

if ! curl -fL -o runsc "$DOWNLOAD_URL" 2>/dev/null; then
  echo "ERROR: Failed to download runsc. Release may not exist." >&2
  echo "       Try: $(basename "$0") --help" >&2
  exit 1
fi

if ! curl -fL -o runsc.sha512 "$CHECKSUM_URL" 2>/dev/null; then
  echo "ERROR: Failed to download checksum" >&2
  exit 1
fi

echo "==> Verifying checksum..."
if ! sha512sum -c runsc.sha512 &>/dev/null; then
  echo "ERROR: Checksum verification failed. Downloaded file may be corrupted." >&2
  exit 1
fi
echo "✓ Checksum verified"

echo "==> Installing runsc to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
chmod +x runsc
cp runsc "$INSTALL_DIR/runsc"

RUNSC_PATH="$INSTALL_DIR/runsc"
echo "✓ gVisor installed: $RUNSC_PATH"
echo ""
"$RUNSC_PATH" --version

# Register with Docker daemon
if [[ "$REGISTER_DOCKER" == true ]]; then
  echo ""
  echo "==> Registering gVisor with Docker daemon..."

  DOCKER_CONFIG_DIR="/etc/docker"
  DOCKER_CONFIG="$DOCKER_CONFIG_DIR/daemon.json"

  mkdir -p "$DOCKER_CONFIG_DIR"

  # Backup existing config
  if [[ -f "$DOCKER_CONFIG" ]]; then
    cp "$DOCKER_CONFIG" "$DOCKER_CONFIG.bak"
    echo "✓ Backed up existing config: $DOCKER_CONFIG.bak"
  fi

  # Create new daemon.json with runsc runtime
  # Preserve existing settings if config exists
  if [[ -f "$DOCKER_CONFIG" ]]; then
    # Add runsc to existing config (naive JSON merge)
    # For production, use jq or similar
    cat "$DOCKER_CONFIG" > "$TMP_DIR/daemon.json.tmp"
    cat > "$DOCKER_CONFIG" <<EOF
{
  "runtimes": {
    "runsc": {
      "path": "$RUNSC_PATH",
      "runtimeArgs": []
    }
  }
}
EOF
    echo "⚠ WARNING: Replaced daemon.json. Manual merging may be needed if you had custom settings." >&2
  else
    cat > "$DOCKER_CONFIG" <<EOF
{
  "runtimes": {
    "runsc": {
      "path": "$RUNSC_PATH",
      "runtimeArgs": []
    }
  }
}
EOF
  fi

  echo "✓ Docker daemon config updated: $DOCKER_CONFIG"

  # Restart Docker
  echo ""
  echo "==> Restarting Docker daemon..."

  # Detect init system
  if command -v systemctl &> /dev/null; then
    systemctl restart docker
    echo "✓ Docker restarted (systemctl)"
  elif command -v service &> /dev/null; then
    service docker restart
    echo "✓ Docker restarted (service)"
  else
    echo "⚠ Could not auto-restart Docker. Restart manually:"
    echo "   sudo systemctl restart docker"
  fi

  # Verify registration
  sleep 2
  echo ""
  echo "==> Verifying gVisor is registered..."

  if ! docker ps &>/dev/null; then
    echo "⚠ WARNING: Could not connect to Docker daemon" >&2
  else
    # Try running a simple container with runsc
    echo "Testing gVisor with alpine container..."
    if docker run --rm --runtime=runsc alpine uname -a 2>/dev/null | grep -q gVisor; then
      echo "✓ gVisor is working!"
    elif docker run --rm --runtime=runsc alpine uname -a &>/dev/null; then
      echo "✓ gVisor registered (verify with: docker run --rm --runtime=runsc alpine uname -a)"
    else
      echo "⚠ Warning: Could not test gVisor. Check logs:"
      echo "   sudo journalctl -u docker -n 20"
    fi
  fi
fi

echo ""
echo "==> Installation complete!"
echo ""
echo "Next steps:"
echo "  1. Verify installation: runsc --version"
echo "  2. For Capstone worker, set: WORKER_DOCKER_RUNTIME=runsc"
echo "  3. See docs/GVISOR.md for configuration details"
echo ""
