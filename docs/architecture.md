# Lambdaモノレポ アーキテクチャ

## 全体構成

```text
GitHub
  |
  | CodeConnections
  v
CodePipeline
  |
  +-- Source: 対象コミットを取得
  |
  +-- Build / Deploy: CodeBuild
        |-- Git差分を取得
        |-- Go依存グラフから影響対象を判定
	    |-- 対象ごとにarm64コンテナイメージをビルド
	    |-- 関数別ECR Repositoryへpush
        |-- Lambda関数コードを更新してVersionを発行
        `-- CodeDeploy Deploymentを開始

CodeDeploy: LambdaAllAtOnce
  `-- live Aliasを旧Versionから新Versionへ100%切り替え

API Gateway HTTP API
  |-- /users/*  -> Lambda: user:live
  |-- /orders/* -> Lambda: order:live
  `-- /posts/*  -> Lambda: post:live

CloudWatch Logs <- Lambda / API Gateway
SSM Parameter Store <- 最後に正常処理したGitコミットSHA
```

## 採用方針

### Lambdaパッケージ

関数ごとのECRコンテナイメージを使用します。`cmd/user`、`cmd/order`、`cmd/post`をそれぞれLinux arm64向けの`bootstrap`としてビルドし、Lambdaのコンテナランタイムで実行します。

```bash
docker build --platform linux/arm64 \
  --build-arg FUNCTION_NAME=user \
  -f build/lambda.Dockerfile .
```

基本アーキテクチャは`arm64`とします。ネイティブライブラリを導入する場合は、arm64対応とAmazon Linux 2023での動作を個別に確認します。

### API Gateway

公開APIにはHTTP APIを使用します。REST API固有の機能が必要になった場合だけ移行を検討します。

- API Gatewayは環境ごとに1つ作成する。
- ルートとLambdaの対応はCDKで明示する。
- Lambda関数URLは使用しない。
- 本番環境ではカスタムドメイン、TLS、認証、アクセスログを追加検討する。

### ネットワーク

初期構成のLambdaはVPCへ配置しません。インターネット向けAWS APIや外部APIだけを利用する関数では、VPC、NAT Gateway、ALBの固定費と運用を避けられます。

RDSやElastiCacheなどVPC内リソースへの接続が必要になった関数だけ、VPC接続を別途設計します。VPC接続を導入する際は、NAT Gatewayの要否、VPC Endpoint、Security Group、同時実行数によるDB接続数を確認します。

## インフラ管理

AWS CDK for Goで次を管理します。

- Lambda関数、Version、`live` Alias、実行ロール、CloudWatch Logs保持期間
- API Gateway HTTP API、Integration、Route、Stage
- CodeDeploy Applicationと関数ごとのDeployment Group
- 関数ごとのECR Repository
- CodeConnectionsをSourceとするCodePipeline
- 変更判定とデプロイを実行するCodeBuild
- 最終正常処理コミットを保存するSSM Parameter
- 必要最小限のIAM Policy

アプリケーションコードの通常更新では、CodeBuildが対象イメージをECRへpushし、`UpdateFunctionCode`とVersion発行を行い、CodeDeployが`live` Aliasを新Versionへ切り替えます。関数追加、Route変更、メモリ、タイムアウト、環境変数、IAMなど構成変更はCDKで反映します。

## All-at-onceデプロイ

各関数の旧VersionをBlue、新VersionをGreenとして扱います。`CodeDeployDefault.LambdaAllAtOnce`は、検証開始後に`live` Aliasの通信を新Versionへ一度に100%切り替えます。ECS Cluster、Task Set、ALB、Target Groupは使用しません。

```text
変更前: API Gateway -> live Alias -> Version 1 (100%)
変更後: API Gateway -> live Alias -> Version 2 (100%)
```

初期構成では仕組みを簡潔に保つため、PreTraffic／PostTraffic HookとCloudWatch Alarmは必須にしません。CodeDeployの導入により、デプロイ履歴、VersionとAliasを使った切り替え、将来の自動ロールバックやCanaryへの拡張経路を確保します。

## セキュリティ

- Lambda実行ロールは関数ごとに分離する。
- CodeBuildには対象Lambdaのコード更新とSSM Parameter操作に必要な権限だけを付与する。
- SecretをGitやLambda環境変数へ平文保存せず、Secrets ManagerまたはSSM Parameter Storeを利用する。
- API認証が必要なRouteにはJWT Authorizerなどを設定する。
- API GatewayとLambdaのログに認証情報や個人情報を出力しない。

## コスト上の考え方

LambdaとHTTP APIはリクエスト量に応じて課金され、アイドル時の常駐タスクやALBを不要にできます。一方、高トラフィック、長時間処理、常時高負荷ではECS等の方が有利になる可能性があるため、CloudWatchの実測値をもとに再評価します。
