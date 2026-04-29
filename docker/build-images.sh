#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOCKER_BIN="${DOCKER:-docker}"
IMAGE_PREFIX="${IMAGE_PREFIX:-capstone}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

usage() {
  cat <<EOF
Usage: $(basename "$0") [--prefix PREFIX] [--tag TAG]

Build every Docker image described by subdirectories in:
  ${SCRIPT_DIR}

Each image is tagged as:
  PREFIX-<directory-name>:TAG

Environment overrides:
  DOCKER        Docker executable to use (default: docker)
  IMAGE_PREFIX  Image name prefix (default: capstone)
  IMAGE_TAG     Image tag (default: latest)
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)
      IMAGE_PREFIX="${2:?missing value for --prefix}"
      shift 2
      ;;
    --tag)
      IMAGE_TAG="${2:?missing value for --tag}"
      shift 2
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

found=0

for dockerfile in "${SCRIPT_DIR}"/*/Dockerfile; do
  [[ -f "${dockerfile}" ]] || continue

  found=1
  context_dir="$(dirname "${dockerfile}")"
  image_name="${IMAGE_PREFIX}-$(basename "${context_dir}"):${IMAGE_TAG}"

  printf '==> Building %s from %s\n' "${image_name}" "${context_dir}"
  "${DOCKER_BIN}" build -t "${image_name}" "${context_dir}"
done

if [[ "${found}" -eq 0 ]]; then
  echo "no Dockerfiles found under ${SCRIPT_DIR}" >&2
  exit 1
fi
