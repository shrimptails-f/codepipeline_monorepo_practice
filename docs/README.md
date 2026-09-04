# ドキュメント

本ディレクトリは、Goモノレポ内の複数アプリケーションをAWS Lambdaへ選択的にデプロイする構成を説明します。

> [!NOTE]
> 本文書は移行後の目標設計です。現在のアプリケーションコードとCDKは、この設計に沿って今後移行します。

## 目標

- `cmd/user`、`cmd/order`、`cmd/post`を、それぞれ独立したLambda関数として管理する。
- 共通コードは`internal`に置き、アプリケーションの責務を崩さない。
- Git差分とGoの依存グラフから影響を受ける関数だけを判定する。
- 影響対象だけをビルドし、ZIP形式でLambdaへデプロイする。
- Lambda Versionと`live` Aliasを使い、CodeDeployのAll-at-onceで全通信を新Versionへ切り替える。
- 1つのAmazon API Gateway HTTP APIから各Lambda関数を公開する。
- 常駐コンテナ、ALB、ECRを使わず、小規模環境の固定費を抑える。

## 文書一覧

| 文書 | 内容 |
| --- | --- |
| [architecture.md](architecture.md) | AWS全体構成と設計判断 |
| [directory-structure.md](directory-structure.md) | 目標ディレクトリ構成と責務 |
| [change-detection.md](change-detection.md) | 変更影響を受けるLambdaの判定仕様 |
| [pipeline-design.md](pipeline-design.md) | CodePipelineとCodeBuildの設計 |
| [deployment.md](deployment.md) | 初回構築、通常デプロイ、動作確認 |
| [troubleshooting.md](troubleshooting.md) | 障害の切り分け |

## 命名規則

- リポジトリ／ローカルディレクトリ: `codepipeline_monorepo_practice`
- AWSリソースの共通プレフィックス: `<env>-codepipeline-monorepo-practice`
- 環境名: `dev`、`stg`、`prod`
- Lambda関数: `<env>-codepipeline-monorepo-practice-<function-name>`
- CDK Stack: `<env>-codepipeline-monorepo-practice-<purpose>-stack`

## 管理するLambda

| cmd | Lambda名 | HTTP API Route |
| --- | --- | --- |
| `cmd/user` | `<env>-codepipeline-monorepo-practice-user` | `/users`、`/users/{proxy+}` |
| `cmd/order` | `<env>-codepipeline-monorepo-practice-order` | `/orders`、`/orders/{proxy+}` |
| `cmd/post` | `<env>-codepipeline-monorepo-practice-post` | `/posts`、`/posts/{proxy+}` |

文書中のAWSアカウントID、ARN、接続ID、API URLはプレースホルダーで表記し、実値をコミットしません。
