package stacks

import (
	_jsii_ "github.com/aws/jsii-runtime-go"

	awscdk "github.com/aws/aws-cdk-go/awscdk/v2"
	awsecr "github.com/aws/aws-cdk-go/awscdk/v2/awsecr"
	awss3 "github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	awsssm "github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/shrimptails-f/codepipeline_practice/infra/common"
	"github.com/shrimptails-f/codepipeline_practice/infra/config"
)

type StorageStack struct {
	awscdk.Stack
	ArtifactBucket awss3.Bucket
	Repositories   map[string]awsecr.Repository
}

func NewStorageStack(scope constructs.Construct, cfg config.Config, props *awscdk.StackProps) *StorageStack {
	stack := awscdk.NewStack(scope, _jsii_.String(common.ResourceName(cfg.Environment, "storage-stack")), props)

	bucket := awss3.NewBucket(stack, _jsii_.String("PipelineArtifacts"), &awss3.BucketProps{
		BlockPublicAccess: awss3.BlockPublicAccess_BLOCK_ALL(),
		Encryption:        awss3.BucketEncryption_S3_MANAGED,
		EnforceSSL:        _jsii_.Bool(true),
		RemovalPolicy:     awscdk.RemovalPolicy_DESTROY,
		AutoDeleteObjects: _jsii_.Bool(true),
		Versioned:         _jsii_.Bool(true),
	})

	repositories := make(map[string]awsecr.Repository, len(common.FunctionNames))
	for _, name := range common.FunctionNames {
		repository := awsecr.NewRepository(stack, _jsii_.String(title(name)+"Repository"), &awsecr.RepositoryProps{
			RepositoryName:     _jsii_.String(common.ResourceName(cfg.Environment, name)),
			ImageScanOnPush:    _jsii_.Bool(true),
			ImageTagMutability: awsecr.TagMutability_IMMUTABLE,
			RemovalPolicy:      awscdk.RemovalPolicy_DESTROY,
			EmptyOnDelete:      _jsii_.Bool(true),
		})
		awscdk.NewCfnOutput(stack, _jsii_.String(title(name)+"RepositoryUri"), &awscdk.CfnOutputProps{
			Value: repository.RepositoryUri(),
		})
		repositories[name] = repository
		awsssm.NewStringParameter(stack, _jsii_.String(title(name)+"ImageTagParameter"), &awsssm.StringParameterProps{
			ParameterName: _jsii_.String(common.ImageTagParameterName(cfg.Environment, name)),
			StringValue:   _jsii_.String(common.UnsetParameterValue),
			Tier:          awsssm.ParameterTier_STANDARD,
		})
	}

	awsssm.NewStringParameter(stack, _jsii_.String("LastSuccessfulCommitParameter"), &awsssm.StringParameterProps{
		ParameterName: _jsii_.String(common.LastSuccessfulCommitParameterName(cfg.Environment)),
		StringValue:   _jsii_.String(common.UnsetParameterValue),
		Tier:          awsssm.ParameterTier_STANDARD,
	})

	return &StorageStack{
		Stack:          stack,
		ArtifactBucket: bucket,
		Repositories:   repositories,
	}
}
