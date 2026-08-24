#!/bin/sh
# Bundles everything a cluster needs into dist/install.yaml, with the image
# pinned to the given version:
#   hack/build-installer.sh v0.1.0
set -eu

VERSION=${1:?usage: hack/build-installer.sh <version>}
IMAGE="ghcr.io/mohammad9227/tenant-controller:${VERSION}"
ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT="$ROOT/dist/install.yaml"

mkdir -p "$ROOT/dist"
: > "$OUT"

append() {
  cat "$1" >> "$OUT"
  printf '\n---\n' >> "$OUT"
}

append "$ROOT/config/deploy/namespace.yaml"
append "$ROOT/config/crd/tenant.yaml"
append "$ROOT/config/rbac.yaml"
sed "s|IMAGE_PLACEHOLDER|$IMAGE|" "$ROOT/config/deploy/operator.yaml" >> "$OUT"

echo "wrote $OUT (image $IMAGE)"
