#!/usr/bin/env bash
# build-rootfs.sh — Build a Firecracker ext4 rootfs image for one language.
#
# Usage:
#   ./build-rootfs.sh <language> <output-dir>
#   ./build-rootfs.sh python /var/lib/fc/rootfs
#
# Requires: docker, mkfs.ext4 (e2fsprogs), root or passwordless sudo.
set -euo pipefail

LANG=${1:?usage: $0 <language> <output-dir>}
OUTPUT_DIR=${2:?usage: $0 <language> <output-dir>}
REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
DOCKERFILE="$REPO_ROOT/docker/firecracker/$LANG/Dockerfile"
IMAGE_TAG="capstone-fc-$LANG:latest"
OUTPUT_FILE="$OUTPUT_DIR/$LANG.ext4"
ROOTFS_SIZE_MB=${ROOTFS_SIZE_MB:-1024}

echo "==> Building Docker image for language: $LANG"
docker build \
  --platform linux/amd64 \
  -f "$DOCKERFILE" \
  -t "$IMAGE_TAG" \
  "$REPO_ROOT"

echo "==> Exporting image filesystem"
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

CONTAINER_ID=$(docker create --platform linux/amd64 "$IMAGE_TAG")
docker export "$CONTAINER_ID" | tar -x -C "$TMP_DIR"
docker rm "$CONTAINER_ID" > /dev/null

echo "==> Creating ext4 image (${ROOTFS_SIZE_MB} MiB)"
mkdir -p "$OUTPUT_DIR"
dd if=/dev/zero of="$OUTPUT_FILE" bs=1M count="$ROOTFS_SIZE_MB" status=progress
mkfs.ext4 -d "$TMP_DIR" "$OUTPUT_FILE"

echo "==> Rootfs written to $OUTPUT_FILE"
du -sh "$OUTPUT_FILE"
