#!/usr/bin/env bash

# --- begin runfiles.bash initialization v3 ---
# Copy-pasted from the Bazel Bash runfiles library v3.
set -uo pipefail; set +e; f=bazel_tools/tools/bash/runfiles/runfiles.bash
# shellcheck disable SC1090
source "${RUNFILES_DIR:-/dev/null}/$f" 2>/dev/null || \
  # shellcheck disable SC1090
  source "$(grep -sm1 "^$f " "${RUNFILES_MANIFEST_FILE:-/dev/null}" | cut -f2- -d' ')" 2>/dev/null || \
  # shellcheck disable SC1090
  source "$0.runfiles/$f" 2>/dev/null || \
  # shellcheck disable SC1090
  source "$(grep -sm1 "^$f " "$0.runfiles_manifest" | cut -f2- -d' ')" 2>/dev/null || \
  # shellcheck disable SC1090
  source "$(grep -sm1 "^$f " "$0.exe.runfiles_manifest" | cut -f2- -d' ')" 2>/dev/null || \
  { echo>&2 "ERROR: cannot find $f"; exit 1; }; f=; set -e
# --- end runfiles.bash initialization v3 ---

## Setting up tools
gcloud=$(rlocation sourcegraph_workspace/dev/tools/gcloud)
packer=$(rlocation sourcegraph_workspace/dev/tools/packer)
base="cmd/executor/docker-mirror/"

## Setting up the folder we're going to use with packer
mkdir workdir
trap "rm -Rf workdir" EXIT

cp $base/docker-mirror.pkr.hcl workdir/
cp $base/aws_regions.json workdir/
cp $base/install.sh workdir/

$gcloud secrets versions access latest --secret=e2e-builder-sa-key --quiet --project=sourcegraph-ci >"workdir/builder-sa-key.json"

## Setting up packer
export PKR_VAR_name
PKR_VAR_name="${IMAGE_FAMILY}-${BUILDKITE_BUILD_NUMBER}"
export PKR_VAR_image_family="${IMAGE_FAMILY}"
export PKR_VAR_tagged_release="${EXECUTOR_IS_TAGGED_RELEASE}"
export PKR_VAR_aws_access_key=${AWS_EXECUTOR_AMI_ACCESS_KEY}
export PKR_VAR_aws_secret_key=${AWS_EXECUTOR_AMI_SECRET_KEY}
# This should prevent some occurrences of Failed waiting for AMI failures:
# https://austincloud.guru/2020/05/14/long-running-packer-builds-failing/
export PKR_VAR_aws_max_attempts=480
export PKR_VAR_aws_poll_delay_seconds=5

cd workdir

export PKR_VAR_aws_regions
if [ "${EXECUTOR_IS_TAGGED_RELEASE}" = "true" ]; then
  PKR_VAR_aws_regions="$(jq -r '.' <aws_regions.json)"
else
  PKR_VAR_aws_regions='["us-west-2"]'
fi

$packer init docker-mirror.pkr.hcl
$packer build -force docker-mirror.pkr.hcl
