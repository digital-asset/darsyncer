#! /usr/bin/env bash
# Copyright 2026 Copyright (c) 2026 Digital Asset (Switzerland) GmbH and/or its affiliates. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
cd "${script_dir}/.."

tag="$(get-snapshot-version)"

export KO_DEFAULTBASEIMAGE="europe-docker.pkg.dev/da-images/public/docker/da-base-image:minimal"

params=()
params+=(--image-label=org.opencontainers.image.base.name="${KO_DEFAULTBASEIMAGE}")
params+=(--image-label=org.opencontainers.image.vendor="Digital Asset")
params+=(--image-label=org.opencontainers.image.authors="Digital Asset Team")
params+=(--image-label=org.opencontainers.image.created=$(date +%FT%T.%3NZ))

params+=(--image-annotation=org.opencontainers.image.base.name="${KO_DEFAULTBASEIMAGE}")
params+=(--image-annotation=org.opencontainers.image.vendor="Digital Asset")
params+=(--image-annotation=org.opencontainers.image.authors="Digital Asset Team")
params+=(--image-annotation=org.opencontainers.image.created=$(date +%FT%T.%3NZ))

# if we aren't running in CI, set the local flag for testing
if [ -z "${GITHUB_ACTION:-}" ]; then
    params+=(--local)
else
    params+=(--image-label=org.opencontainers.image.source=$(sed -E "s/^git@(.*)\:(.*).git/https:\/\/\1\/\2/g" <<< "$GITHUB_SERVER_URL/$GITHUB_REPOSITORY"))
    params+=(--image-label=org.opencontainers.image.url=$(sed -E "s/^git@(.*)\:(.*).git/https:\/\/\1\/\2/g" <<< "$GITHUB_SERVER_URL/$GITHUB_REPOSITORY"))
    params+=(--image-label=org.opencontainers.image.revision="$GITHUB_SHA")
    params+=(--image-annotation=org.opencontainers.image.source=$(sed -E "s/^git@(.*)\:(.*).git/https:\/\/\1\/\2/g" <<< "$GITHUB_SERVER_URL/$GITHUB_REPOSITORY"))
    params+=(--image-annotation=org.opencontainers.image.url=$(sed -E "s/^git@(.*)\:(.*).git/https:\/\/\1\/\2/g" <<< "$GITHUB_SERVER_URL/$GITHUB_REPOSITORY"))
    params+=(--image-annotation=org.opencontainers.image.revision="$GITHUB_SHA")
fi

function build() (
    set -euo pipefail
    name="$1"

    ref_name=()
    ref_name+=(--image-label=org.opencontainers.image.base.ref.name="${KO_DEFAULTBASEIMAGE}")
    ref_name+=(--image-annotation=org.opencontainers.image.base.ref.name="${KO_DEFAULTBASEIMAGE}")

    set -x
#    KO_DOCKER_REPO="${reg}/${name}" \
        ko build \
        --platform="linux/amd64,linux/arm64" \
        "${params[@]}" \
        "${ref_name[@]}" \
        --bare \
        --sbom spdx \
        --tags "${tag}" \
        "./cmd/${name}"
)

build "darsyncer"
