package stacks

import (
	"fmt"

	_jsii_ "github.com/aws/jsii-runtime-go"

	awscdk "github.com/aws/aws-cdk-go/awscdk/v2"
	awsapigatewayv2 "github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	awscodebuild "github.com/aws/aws-cdk-go/awscdk/v2/awscodebuild"
	awscodedeploy "github.com/aws/aws-cdk-go/awscdk/v2/awscodedeploy"
	awscodepipeline "github.com/aws/aws-cdk-go/awscdk/v2/awscodepipeline"
	awscodepipelineactions "github.com/aws/aws-cdk-go/awscdk/v2/awscodepipelineactions"
	awsecr "github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	awsiam "github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	awslambda "github.com/aws/aws-cdk-go/awscdk/v2/awslambda"
	awslogs "github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	awsssm "github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/shrimptails-f/codepipeline_practice/infra/common"
	"github.com/shrimptails-f/codepipeline_practice/infra/config"
)

type ApplicationStack struct {
	awscdk.Stack
	Pipeline awscodepipeline.Pipeline
}

func NewApplicationStack(scope constructs.Construct, cfg config.Config, network *NetworkStack, storage *StorageStack, props *awscdk.StackProps) *ApplicationStack {
	stack := awscdk.NewStack(scope, _jsii_.String(common.ResourceName(cfg.Environment, "application-stack")), props)
	application := awscodedeploy.NewLambdaApplication(stack, _jsii_.String("CodeDeployApplication"), &awscodedeploy.LambdaApplicationProps{
		ApplicationName: _jsii_.String(common.ResourceName(cfg.Environment, "codedeploy")),
	})

	for _, name := range common.FunctionNames {
		addLambdaService(stack, cfg, network.API, application, storage.Repositories[name], name)
	}

	pipeline := addPipeline(stack, cfg, storage)
	return &ApplicationStack{Stack: stack, Pipeline: pipeline}
}

func addLambdaService(stack awscdk.Stack, cfg config.Config, api awsapigatewayv2.CfnApi, application awscodedeploy.LambdaApplication, repository awsecr.Repository, name string) {
	functionName := common.ResourceName(cfg.Environment, name)
	logGroup := awslogs.NewLogGroup(stack, _jsii_.String(title(name)+"LogGroup"), &awslogs.LogGroupProps{
		LogGroupName:  _jsii_.String("/aws/lambda/" + functionName),
		Retention:     awslogs.RetentionDays_ONE_MONTH,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})
	role := awsiam.NewRole(stack, _jsii_.String(title(name)+"ExecutionRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(_jsii_.String("lambda.amazonaws.com"), nil),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(_jsii_.String("service-role/AWSLambdaBasicExecutionRole")),
		},
	})
	fn := awslambda.NewDockerImageFunction(stack, _jsii_.String(title(name)+"Function"), &awslambda.DockerImageFunctionProps{
		FunctionName: _jsii_.String(functionName),
		Architecture: awslambda.Architecture_ARM_64(),
		Code: awslambda.DockerImageCode_FromEcr(repository, &awslambda.EcrImageCodeProps{
			TagOrDigest: awsssm.StringParameter_ValueForStringParameter(
				stack,
				_jsii_.String(common.ImageTagParameterName(cfg.Environment, name)),
				nil,
			),
		}),
		Role:       role,
		LogGroup:   logGroup,
		MemorySize: _jsii_.Number(128),
		Timeout:    awscdk.Duration_Seconds(_jsii_.Number(10)),
	})
	version := awslambda.NewVersion(stack, _jsii_.String(title(name)+"InitialVersion"), &awslambda.VersionProps{
		Lambda:        fn,
		RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
	})
	alias := awslambda.NewAlias(stack, _jsii_.String(title(name)+"LiveAlias"), &awslambda.AliasProps{
		AliasName: _jsii_.String("live"),
		Version:   version,
	})
	awscodedeploy.NewLambdaDeploymentGroup(stack, _jsii_.String(title(name)+"DeploymentGroup"), &awscodedeploy.LambdaDeploymentGroupProps{
		Application:         application,
		Alias:               alias,
		DeploymentConfig:    awscodedeploy.LambdaDeploymentConfig_ALL_AT_ONCE(),
		DeploymentGroupName: _jsii_.String(common.ResourceName(cfg.Environment, name+"-deployment-group")),
		AutoRollback: &awscodedeploy.AutoRollbackConfig{
			FailedDeployment:  _jsii_.Bool(true),
			StoppedDeployment: _jsii_.Bool(true),
		},
	})

	integration := awsapigatewayv2.NewCfnIntegration(stack, _jsii_.String(title(name)+"Integration"), &awsapigatewayv2.CfnIntegrationProps{
		ApiId:                api.Ref(),
		IntegrationType:      _jsii_.String("AWS_PROXY"),
		IntegrationMethod:    _jsii_.String("POST"),
		IntegrationUri:       alias.FunctionArn(),
		PayloadFormatVersion: _jsii_.String("2.0"),
	})
	for index, route := range []string{
		fmt.Sprintf("ANY /%ss", name),
		fmt.Sprintf("ANY /%ss/{proxy+}", name),
	} {
		awsapigatewayv2.NewCfnRoute(stack, _jsii_.String(fmt.Sprintf("%sRoute%d", title(name), index)), &awsapigatewayv2.CfnRouteProps{
			ApiId:    api.Ref(),
			RouteKey: _jsii_.String(route),
			Target:   awscdk.Fn_Join(_jsii_.String("/"), &[]*string{_jsii_.String("integrations"), integration.Ref()}),
		})
	}
	alias.AddPermission(_jsii_.String("AllowApiGateway"), &awslambda.Permission{
		Action:    _jsii_.String("lambda:InvokeFunction"),
		Principal: awsiam.NewServicePrincipal(_jsii_.String("apigateway.amazonaws.com"), nil),
		SourceArn: stack.FormatArn(&awscdk.ArnComponents{
			Service:      _jsii_.String("execute-api"),
			Resource:     api.Ref(),
			ResourceName: _jsii_.String(fmt.Sprintf("*/*/%ss*", name)),
			ArnFormat:    awscdk.ArnFormat_SLASH_RESOURCE_NAME,
		}),
	})
}

func addPipeline(stack awscdk.Stack, cfg config.Config, storage *StorageStack) awscodepipeline.Pipeline {
	project := awscodebuild.NewPipelineProject(stack, _jsii_.String("DeployProject"), &awscodebuild.PipelineProjectProps{
		ProjectName: _jsii_.String(common.ResourceName(cfg.Environment, "build-deploy")),
		BuildSpec:   awscodebuild.BuildSpec_FromSourceFilename(_jsii_.String("buildspec.yml")),
		Environment: &awscodebuild.BuildEnvironment{
			BuildImage:  awscodebuild.LinuxBuildImage_STANDARD_7_0(),
			ComputeType: awscodebuild.ComputeType_SMALL,
			Privileged:  _jsii_.Bool(true),
		},
		EnvironmentVariables: &map[string]*awscodebuild.BuildEnvironmentVariable{
			"ENV_NAME":       {Value: _jsii_.String(cfg.Environment)},
			"AWS_ACCOUNT_ID": {Value: _jsii_.String(cfg.AccountID)},
			"AWS_REGION":     {Value: _jsii_.String(cfg.Region)},
			"PARAMETER_NAME": {Value: _jsii_.String(common.LastSuccessfulCommitParameterName(cfg.Environment))},
		},
	})
	for _, name := range common.FunctionNames {
		storage.Repositories[name].GrantPullPush(project)
	}
	grantDeploymentPermissions(project, cfg)

	pipeline := awscodepipeline.NewPipeline(stack, _jsii_.String("Pipeline"), &awscodepipeline.PipelineProps{
		PipelineName:     _jsii_.String(common.ResourceName(cfg.Environment, "pipeline")),
		ArtifactBucket:   storage.ArtifactBucket,
		CrossAccountKeys: _jsii_.Bool(false),
		PipelineType:     awscodepipeline.PipelineType_V2,
		ExecutionMode:    awscodepipeline.ExecutionMode_QUEUED,
	})
	sourceOutput := awscodepipeline.NewArtifact(_jsii_.String("SourceOutput"), nil)
	sourceAction := awscodepipelineactions.NewCodeStarConnectionsSourceAction(&awscodepipelineactions.CodeStarConnectionsSourceActionProps{
		ActionName:           _jsii_.String("GitHubSource"),
		ConnectionArn:        _jsii_.String(cfg.ConnectionArn),
		Owner:                _jsii_.String(cfg.RepositoryOwner),
		Repo:                 _jsii_.String(cfg.RepositoryName),
		Branch:               _jsii_.String(cfg.RepositoryBranch),
		Output:               sourceOutput,
		CodeBuildCloneOutput: _jsii_.Bool(true),
	})
	pipeline.AddStage(&awscodepipeline.StageOptions{
		StageName: _jsii_.String("Source"),
		Actions:   &[]awscodepipeline.IAction{sourceAction},
	})
	pipeline.AddStage(&awscodepipeline.StageOptions{
		StageName: _jsii_.String("BuildDeploy"),
		Actions: &[]awscodepipeline.IAction{
			awscodepipelineactions.NewCodeBuildAction(&awscodepipelineactions.CodeBuildActionProps{
				ActionName: _jsii_.String("SelectiveLambdaDeploy"),
				Project:    project,
				Input:      sourceOutput,
				EnvironmentVariables: &map[string]*awscodebuild.BuildEnvironmentVariable{
					"HEAD_SHA": {Value: sourceAction.Variables().CommitId},
				},
			}),
		},
	})
	return pipeline
}

func grantDeploymentPermissions(project awscodebuild.PipelineProject, cfg config.Config) {
	functionArns := make([]*string, 0, len(common.FunctionNames))
	deploymentGroupArns := make([]*string, 0, len(common.FunctionNames))
	for _, name := range common.FunctionNames {
		functionArns = append(functionArns, awscdk.Stack_Of(project).FormatArn(&awscdk.ArnComponents{
			Service:      _jsii_.String("lambda"),
			Resource:     _jsii_.String("function"),
			ResourceName: _jsii_.String(common.ResourceName(cfg.Environment, name) + "*"),
			ArnFormat:    awscdk.ArnFormat_COLON_RESOURCE_NAME,
		}))
		deploymentGroupArns = append(deploymentGroupArns, awscdk.Stack_Of(project).FormatArn(&awscdk.ArnComponents{
			Service:      _jsii_.String("codedeploy"),
			Resource:     _jsii_.String("deploymentgroup"),
			ResourceName: _jsii_.String(common.ResourceName(cfg.Environment, "codedeploy") + "/" + common.ResourceName(cfg.Environment, name+"-deployment-group")),
			ArnFormat:    awscdk.ArnFormat_COLON_RESOURCE_NAME,
		}))
	}
	project.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   _jsii_.Strings("lambda:UpdateFunctionCode", "lambda:GetFunctionConfiguration", "lambda:PublishVersion", "lambda:GetAlias"),
		Resources: &functionArns,
	}))
	project.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions: _jsii_.Strings(
			"codedeploy:CreateDeployment",
			"codedeploy:GetDeployment",
			"codedeploy:GetDeploymentConfig",
			"codedeploy:RegisterApplicationRevision",
		),
		Resources: func() *[]*string {
			resources := append(deploymentGroupArns, _jsii_.String("*"))
			return &resources
		}(),
	}))
	parameterArns := []*string{parameterArn(project, common.LastSuccessfulCommitParameterName(cfg.Environment))}
	for _, name := range common.FunctionNames {
		parameterArns = append(parameterArns, parameterArn(project, common.ImageTagParameterName(cfg.Environment, name)))
	}
	project.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   _jsii_.Strings("ssm:GetParameter", "ssm:PutParameter"),
		Resources: &parameterArns,
	}))
	project.AddToRolePolicy(awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
		Actions:   _jsii_.Strings("codeconnections:UseConnection"),
		Resources: _jsii_.Strings(cfg.ConnectionArn),
	}))
}

func parameterArn(scope constructs.Construct, parameterName string) *string {
	return awscdk.Stack_Of(scope).FormatArn(&awscdk.ArnComponents{
		Service:      _jsii_.String("ssm"),
		Resource:     _jsii_.String("parameter"),
		ResourceName: _jsii_.String(parameterName[1:]),
	})
}

func title(value string) string {
	return string(value[0]-'a'+'A') + value[1:]
}
