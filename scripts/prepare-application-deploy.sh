#!/usr/bin/env bash
set -euo pipefail

readonly ENV_NAME="dev"
readonly PROJECT_NAME="codepipeline-monorepo-practice"
readonly UNSET_VALUE="UNSET"
readonly FUNCTIONS=(user order post)
readonly BASE_PARAMETER="/${ENV_NAME}/${PROJECT_NAME}/last-successful-commit"

if [[ "${1:-}" == "--initialize" ]]; then
  image_tag="${2:-}"
  if [[ ! "${image_tag}" =~ ^[0-9a-f]{7,40}$ ]]; then
    echo "IMAGE_TAG must be a 7-40 character lowercase Git SHA." >&2
    exit 1
  fi

  aws ssm put-parameter --name "${BASE_PARAMETER}" --type String --value "${image_tag}" --overwrite
  for function_name in "${FUNCTIONS[@]}"; do
    aws ssm put-parameter \
      --name "/${ENV_NAME}/${PROJECT_NAME}/functions/${function_name}/image-tag" \
      --type String \
      --value "${image_tag}" \
      --overwrite
  done
fi

parameters=("${BASE_PARAMETER}")
for function_name in "${FUNCTIONS[@]}"; do
  parameters+=("/${ENV_NAME}/${PROJECT_NAME}/functions/${function_name}/image-tag")
done

for parameter_name in "${parameters[@]}"; do
  value="$(aws ssm get-parameter --name "${parameter_name}" --query 'Parameter.Value' --output text)"
  if [[ -z "${value}" || "${value}" == "None" || "${value}" == "${UNSET_VALUE}" ]]; then
    echo "SSM Parameter is not initialized: ${parameter_name}" >&2
    exit 1
  fi
done
