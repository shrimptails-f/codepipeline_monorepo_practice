#!/usr/bin/env bash
set -euo pipefail

: "${ENV_NAME:?ENV_NAME is required}"
: "${AWS_ACCOUNT_ID:?AWS_ACCOUNT_ID is required}"
: "${AWS_REGION:?AWS_REGION is required}"
: "${HEAD_SHA:?HEAD_SHA is required}"
: "${PARAMETER_NAME:?PARAMETER_NAME is required}"

readonly FUNCTIONS=(user order post)
readonly APPLICATION_NAME="${ENV_NAME}-codepipeline-monorepo-practice-codedeploy"

base_sha="$(aws ssm get-parameter --name "${PARAMETER_NAME}" --query 'Parameter.Value' --output text)"
if [[ -z "${base_sha}" || "${base_sha}" == "None" || "${base_sha}" == "UNSET" ]]; then
  echo "Deployment baseline is not initialized: ${PARAMETER_NAME}" >&2
  exit 1
fi

if ! git cat-file -e "${base_sha}^{commit}" 2>/dev/null; then
  git fetch --no-tags origin "${base_sha}" || true
fi

declare -a targets=()
if ! git cat-file -e "${base_sha}^{commit}" 2>/dev/null; then
  echo "Baseline ${base_sha} is unavailable; deploying all functions."
  targets=("${FUNCTIONS[@]}")
else
  mapfile -t changes < <(git diff --name-status "${base_sha}" "${HEAD_SHA}")
  declare -A selected=()
  deploy_all=false

  for change in "${changes[@]}"; do
    status="${change%%$'\t'*}"
    paths="${change#*$'\t'}"
    if [[ "${status}" == D* || "${status}" == R* ]]; then
      if [[ "${paths}" == cmd/* || "${paths}" == internal/* ]]; then
        deploy_all=true
      fi
      continue
    fi

    path="${paths##*$'\t'}"
    case "${path}" in
      docs/*|*_test.go)
        ;;
      cmd/user/*) selected[user]=1 ;;
      cmd/order/*) selected[order]=1 ;;
      cmd/post/*) selected[post]=1 ;;
      internal/*.go|internal/*/*.go|internal/*/*/*.go)
        package_dir="${path%/*}"
        import_path="$(go list -f '{{.ImportPath}}' "./${package_dir}" 2>/dev/null || true)"
        if [[ -z "${import_path}" ]]; then
          deploy_all=true
          continue
        fi
        for function_name in "${FUNCTIONS[@]}"; do
          if go list -deps -f '{{.ImportPath}}' "./cmd/${function_name}" | grep -Fxq "${import_path}"; then
            selected["${function_name}"]=1
          fi
        done
        ;;
      go.mod|go.sum|buildspec.yml|scripts/*|infra/*) deploy_all=true ;;
      *) deploy_all=true ;;
    esac
  done

  if [[ "${deploy_all}" == true ]]; then
    targets=("${FUNCTIONS[@]}")
  else
    for function_name in "${FUNCTIONS[@]}"; do
      [[ -z "${selected[${function_name}]:-}" ]] || targets+=("${function_name}")
    done
  fi
fi

echo "BASE_SHA=${base_sha}"
echo "HEAD_SHA=${HEAD_SHA}"
echo "Deploy targets: ${targets[*]:-(none)}"
go test ./...

registry="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
export DOCKER_BUILDKIT=1
aws ecr get-login-password --region "${AWS_REGION}" |
  docker login --username AWS --password-stdin "${registry}"
build_dir="$(mktemp -d)"
trap 'rm -rf "${build_dir}"' EXIT

for function_name in "${targets[@]}"; do
	function_dir="${build_dir}/${function_name}"
  mkdir -p "${function_dir}"
  repository_name="${ENV_NAME}-codepipeline-monorepo-practice-${function_name}"
  image_uri="${registry}/${repository_name}:${HEAD_SHA}"
  if ! aws ecr describe-images --repository-name "${repository_name}" --image-ids "imageTag=${HEAD_SHA}" >/dev/null 2>&1; then
    docker build --platform linux/arm64 \
      --build-arg "FUNCTION_NAME=${function_name}" \
      --tag "${image_uri}" \
      --file build/lambda.Dockerfile .
    docker push "${image_uri}"
  fi

  lambda_name="${ENV_NAME}-codepipeline-monorepo-practice-${function_name}"
  target_version="$(aws lambda update-function-code --function-name "${lambda_name}" --image-uri "${image_uri}" --publish --query Version --output text)"
  aws lambda wait function-updated --function-name "${lambda_name}"
  current_version="$(aws lambda get-alias --function-name "${lambda_name}" --name live --query FunctionVersion --output text)"

  appspec="${function_dir}/appspec.yml"
  {
    echo "version: 0.0"
    echo "Resources:"
    echo "  - LambdaFunction:"
    echo "      Type: AWS::Lambda::Function"
    echo "      Properties:"
    echo "        Name: ${lambda_name}"
    echo "        Alias: live"
    echo "        CurrentVersion: ${current_version}"
    echo "        TargetVersion: ${target_version}"
  } > "${appspec}"

  revision="${function_dir}/revision.json"
  jq -n --rawfile content "${appspec}" '{revisionType:"AppSpecContent",appSpecContent:{content:$content}}' > "${revision}"
  deployment_id="$(aws deploy create-deployment --application-name "${APPLICATION_NAME}" --deployment-group-name "${ENV_NAME}-codepipeline-monorepo-practice-${function_name}-deployment-group" --revision "file://${revision}" --query deploymentId --output text)"
  aws deploy wait deployment-successful --deployment-id "${deployment_id}"
  deployed_version="$(aws lambda get-alias --function-name "${lambda_name}" --name live --query FunctionVersion --output text)"
  [[ "${deployed_version}" == "${target_version}" ]]
done

for function_name in "${targets[@]}"; do
  aws ssm put-parameter \
    --name "/${ENV_NAME}/codepipeline-monorepo-practice/functions/${function_name}/image-tag" \
    --type String \
    --value "${HEAD_SHA}" \
    --overwrite
done
aws ssm put-parameter --name "${PARAMETER_NAME}" --type String --value "${HEAD_SHA}" --overwrite
