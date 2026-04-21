#!/usr/bin/env bash
set -euo pipefail
# Push to an HTTP registry (typical on a Pi): add REGISTRY host to Docker "insecure-registries" — see README.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -f "${SCRIPT_DIR}/go.mod" ]]; then
  BOT_DIR="${SCRIPT_DIR}"
elif [[ -f "${SCRIPT_DIR}/bot/go.mod" ]]; then
  BOT_DIR="${SCRIPT_DIR}/bot"
else
  echo "deploy.sh must sit in the bot module directory or in the repository root." >&2
  exit 1
fi

REGISTRY="${REGISTRY:-192.168.1.100:5000}"
IMAGE_NAME="${IMAGE_NAME:-lifesoundtrack-bot}"
TAG="${TAG:-latest}"
# Raspberry Pi 64-bit: linux/arm64. Multi-arch example: PLATFORM=linux/amd64,linux/arm64
PLATFORM="${PLATFORM:-linux/arm64}"
BUILDER="${BUILDER:-lifesoundtrack-buildx}"

REMOTE_IMAGE="${REGISTRY}/${IMAGE_NAME}:${TAG}"

docker buildx create --use --name "${BUILDER}" 2>/dev/null || docker buildx use "${BUILDER}"

echo "Building ${REMOTE_IMAGE} (platform: ${PLATFORM}, context: ${BOT_DIR})"

# --push from buildx uses BuildKit, which often ignores the engine's "insecure-registries"
# and still talks HTTPS to a plain-HTTP registry. --load + docker push uses the daemon, so
# insecure-registries apply. --load is only valid for a single platform.
if [[ "${PLATFORM}" == *","* ]]; then
  echo "Multi-arch: pushing via buildx (see README if HTTP registry push fails)" >&2
  docker buildx build \
    --platform "${PLATFORM}" \
    -f "${BOT_DIR}/Dockerfile" \
    -t "${REMOTE_IMAGE}" \
    --push \
    "${BOT_DIR}"
else
  docker buildx build \
    --platform "${PLATFORM}" \
    -f "${BOT_DIR}/Dockerfile" \
    -t "${REMOTE_IMAGE}" \
    --load \
    "${BOT_DIR}"
  echo "Pushing ${REMOTE_IMAGE} (via Docker engine)"
  docker push "${REMOTE_IMAGE}"
fi

echo "Done. On the Pi: docker pull ${REMOTE_IMAGE}"
