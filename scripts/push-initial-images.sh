#!/usr/bin/env bash
set -euo pipefail

readonly AWS_ACCOUNT_ID="654654388040"
readonly AWS_REGION="ap-northeast-1"
readonly ENV_NAME="dev"
readonly PROJECT_NAME="codepipeline-monorepo-practice"
readonly FUNCTIONS=(user order post)

image_tag="${1:-}"
if [[ ! "${image_tag}" =~ ^[0-9a-f]{7,40}$ ]]; then
  echo "IMAGE_TAG must be a 7-40 character lowercase Git SHA." >&2
  exit 1
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "docker command is required to build and push Lambda images." >&2
  exit 1
fi

registry="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
export DOCKER_BUILDKIT=1
aws ecr get-login-password --region "${AWS_REGION}" |
  docker login --username AWS --password-stdin "${registry}"

for function_name in "${FUNCTIONS[@]}"; do
  repository_name="${ENV_NAME}-${PROJECT_NAME}-${function_name}"
  image_uri="${registry}/${repository_name}:${image_tag}"

  if aws ecr describe-images \
    --repository-name "${repository_name}" \
    --image-ids "imageTag=${image_tag}" >/dev/null 2>&1; then
    echo "Image already exists, skipping: ${image_uri}"
    continue
  fi

  docker build --platform linux/arm64 \
    --build-arg "FUNCTION_NAME=${function_name}" \
    --tag "${image_uri}" \
    --file build/lambda.Dockerfile .
  docker push "${image_uri}"
done
