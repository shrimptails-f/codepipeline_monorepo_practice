package common

import "fmt"

const (
	ProjectName            = "codepipeline_monorepo_practice"
	ProjectResourceName    = "codepipeline-monorepo-practice"
	DeployEnvironment      = "dev"
	AWSAccountID           = "654654388040"
	AWSRegion              = "ap-northeast-1"
	CodeConnectionArn      = "arn:aws:codeconnections:ap-northeast-1:654654388040:connection/6c4de5b9-0e18-4e85-85e4-bef6fc2453c8"
	GitHubRepositoryURL    = "https://github.com/shrimptails-f/codepipeline_monorepo_practice"
	GitHubRepositoryOwner  = "shrimptails-f"
	GitHubRepositoryName   = "codepipeline_monorepo_practice"
	GitHubRepositoryBranch = "develop"
	UnsetParameterValue    = "UNSET"
)

var FunctionNames = []string{"user", "order", "post"}

func ResourceName(environment, resource string) string {
	return fmt.Sprintf("%s-%s-%s", environment, ProjectResourceName, resource)
}

func ExportName(environment, resource string) string {
	return ResourceName(environment, resource)
}

func LastSuccessfulCommitParameterName(environment string) string {
	return fmt.Sprintf("/%s/%s/last-successful-commit", environment, ProjectResourceName)
}

func ImageTagParameterName(environment, functionName string) string {
	return fmt.Sprintf("/%s/%s/functions/%s/image-tag", environment, ProjectResourceName, functionName)
}
