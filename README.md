# Lambdaモノレポ CodePipeline

Goモノレポ内の`user`、`order`、`post`を独立したAWS Lambdaとして管理し、Git差分とGo依存グラフから影響を受ける関数だけをCodePipelineでデプロイします。

## インフラ構成

`infra/main.go`を唯一のCDKエントリポイントとし、次の3層を同時に管理します。

- Network Stack: API Gateway HTTP API
- Storage Stack: CodePipelineアーティファクト用S3 Bucket、関数別の3 ECR Repository
- Application Stack: ECRイメージ形式のLambda、Version、`live` Alias、CodeDeploy、CodeBuild、CodePipeline

Lambdaは`arm64`のコンテナイメージ形式です。関数ごとに専用のECR Repositoryを使用します。

## ローカル確認

```bash
task test
task infra:test
task infra:synth
```

初回デプロイは次の3段階です。

1. `task deploy:stack`でNetwork Stack、Storage Stack、SSM Parameterを作成する。
2. `task image:push IMAGE_TAG=<Git SHA>`で各ECR Repositoryへarm64イメージをpushする。
3. `task deploy:application:init IMAGE_TAG=<Git SHA>`でSSMを初期化し、Application Stackを作成する。

2回目以降のApplication Stackデプロイは`task deploy:application`を使用し、関数ごとのイメージタグをSSMから参照します。

詳細は[docs/README.md](docs/README.md)を参照してください。
