package stacks

import (
	_jsii_ "github.com/aws/jsii-runtime-go"

	awscdk "github.com/aws/aws-cdk-go/awscdk/v2"
	awsapigatewayv2 "github.com/aws/aws-cdk-go/awscdk/v2/awsapigatewayv2"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/shrimptails-f/codepipeline_practice/infra/common"
	"github.com/shrimptails-f/codepipeline_practice/infra/config"
)

type NetworkStack struct {
	awscdk.Stack
	API awsapigatewayv2.CfnApi
}

// 実務を見据えてnetworkスタックを用意してます。
// このくらいの実装なら本来は分ける必要はありませんが、学習用です。
func NewNetworkStack(scope constructs.Construct, cfg config.Config, props *awscdk.StackProps) *NetworkStack {
	stack := awscdk.NewStack(scope, _jsii_.String(common.ResourceName(cfg.Environment, "network-stack")), props)

	api := awsapigatewayv2.NewCfnApi(stack, _jsii_.String("HttpApi"), &awsapigatewayv2.CfnApiProps{
		Name:         _jsii_.String(common.ResourceName(cfg.Environment, "api")),
		ProtocolType: _jsii_.String("HTTP"),
	})
	awsapigatewayv2.NewCfnStage(stack, _jsii_.String("DefaultStage"), &awsapigatewayv2.CfnStageProps{
		ApiId:      api.Ref(),
		StageName:  _jsii_.String("$default"),
		AutoDeploy: _jsii_.Bool(true),
	})

	awscdk.NewCfnOutput(stack, _jsii_.String("ApiEndpoint"), &awscdk.CfnOutputProps{
		Value: api.AttrApiEndpoint(),
	})

	return &NetworkStack{Stack: stack, API: api}
}
