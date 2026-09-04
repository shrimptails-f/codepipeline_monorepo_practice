package main

import (
	"log"

	_jsii_ "github.com/aws/jsii-runtime-go"

	awscdk "github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/shrimptails-f/codepipeline_practice/infra/config"
	"github.com/shrimptails-f/codepipeline_practice/infra/stacks"
)

func main() {
	defer _jsii_.Close()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	app := awscdk.NewApp(nil)
	props := stackProps(cfg)

	network := stacks.NewNetworkStack(app, cfg, props)
	storage := stacks.NewStorageStack(app, cfg, props)
	stacks.NewApplicationStack(app, cfg, network, storage, props)

	app.Synth(nil)
}

func stackProps(cfg config.Config) *awscdk.StackProps {
	if cfg.AccountID == "" || cfg.Region == "" {
		return nil
	}

	return &awscdk.StackProps{
		Env: &awscdk.Environment{
			Account: _jsii_.String(cfg.AccountID),
			Region:  _jsii_.String(cfg.Region),
		},
	}
}
