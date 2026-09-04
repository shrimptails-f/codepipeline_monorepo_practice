package config

import (
	"fmt"

	"github.com/shrimptails-f/codepipeline_practice/infra/common"
)

type Config struct {
	Environment      string
	AccountID        string
	Region           string
	ConnectionArn    string
	RepositoryOwner  string
	RepositoryName   string
	RepositoryBranch string
}

func Load() (Config, error) {
	cfg := Config{
		Environment:      common.DeployEnvironment,
		AccountID:        common.AWSAccountID,
		Region:           common.AWSRegion,
		ConnectionArn:    common.CodeConnectionArn,
		RepositoryOwner:  common.GitHubRepositoryOwner,
		RepositoryName:   common.GitHubRepositoryName,
		RepositoryBranch: common.GitHubRepositoryBranch,
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Environment == "" || c.AccountID == "" || c.Region == "" {
		return fmt.Errorf("deployment environment, account ID, and region are required")
	}
	if c.ConnectionArn == "" {
		return fmt.Errorf("code connection ARN is required")
	}
	if c.RepositoryOwner == "" || c.RepositoryName == "" || c.RepositoryBranch == "" {
		return fmt.Errorf("github repository owner, name, and branch are required")
	}
	return nil
}
