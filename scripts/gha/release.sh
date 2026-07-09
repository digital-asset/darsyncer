#!/usr/bin/env bash
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
cd "${script_dir}/../.."


version="$(<VERSION)"
snap="$(get-snapshot-version)"

function publish() {
  snapshot="europe-docker.pkg.dev/da-images/public-unstable/docker"
  public_release="europe-docker.pkg.dev/da-images/public/docker"
  
  snap_image="${snapshot}/${1}:${snap}"
  release_image="${public_release}/${1}:${version}"

  if gcrane manifest "${release_image}" &>/dev/null; then
    echo "skipping: ${release_image}"
    return
  fi

  echo "promoting: ${release_image}"
  gcrane cp "${snap_image}" "${release_image}"

  echo "tag latest"
  gcrane tag "${release_image}" "latest"
}

publish "darsyncer"
