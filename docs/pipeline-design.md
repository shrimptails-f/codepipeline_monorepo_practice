# CodePipeline設計

## 目的

GitHubの対象ブランチへのpushを起点に、影響を受けるGo Lambdaだけをテスト、ビルド、デプロイします。

## 基本設定

| 項目 | 値 |
| --- | --- |
| Pipeline名 | `<env>-codepipeline-monorepo-practice-pipeline` |
| Source | GitHub（CodeConnections） |
| 対象Repository | `<owner>/codepipeline_monorepo_practice` |
| 対象Branch | 環境設定で指定（例: `develop`） |
| Build | CodeBuild |
| Lambda Runtime | `provided.al2023` |
| Architecture | `arm64` |
| Package | ECRコンテナイメージ |
| API | API Gateway HTTP API |
| Alias | `live` |
| Deploy | CodeDeploy `LambdaAllAtOnce` |

## デプロイ対象

| 判定名 | Build package | Lambda関数名 |
| --- | --- | --- |
| `user` | `./cmd/user` | `<env>-codepipeline-monorepo-practice-user` |
| `order` | `./cmd/order` | `<env>-codepipeline-monorepo-practice-order` |
| `post` | `./cmd/post` | `<env>-codepipeline-monorepo-practice-post` |

## ステージ

### Source

CodeConnectionsでGitHubから対象コミットを取得します。変更判定にGit履歴が必要なため、CodeBuildがGitメタデータへアクセスできるClone形式の出力を使用します。

Source ActionのCommit IDを後続Actionへ渡し、`HEAD_SHA`として固定します。Pipeline実行中にBranchが進んでも、別コミットを誤ってデプロイしないようにします。

### Build / Deploy

CodeBuildで次を順に実行します。

SSM Parameterと初期イメージはApplication Stackの初回デプロイ前に準備済みとします。Parameterが存在しない場合や`UNSET`の場合、Pipelineは異常終了します。

1. SSM Parameterから`BASE_SHA`を取得する。
2. `BASE_SHA..HEAD_SHA`の変更ファイルを取得する。
3. 影響を受ける関数を判定する。
4. 対象パッケージのテストを実行する。
5. 対象ごとにarm64コンテナイメージをビルドし、専用ECR Repositoryへpushする。
6. `aws lambda update-function-code --image-uri ... --publish`で対象関数を更新し、新Versionを発行する。
7. 現在`live` Aliasが参照するVersionと新VersionからLambda用AppSpecを生成する。
8. 対象関数のCodeDeploy Deployment GroupでAll-at-onceデプロイを開始する。
9. CodeDeployの成功を待ち、`live` Aliasが新Versionを参照したことを確認する。
10. 最後にSSM Parameterを`HEAD_SHA`へ更新する。

```bash
aws lambda update-function-code \
  --function-name "${ENV_NAME}-codepipeline-monorepo-practice-${FUNCTION_NAME}" \
  --image-uri "${IMAGE_URI}" \
  --publish
```

AppSpecには関数名、Alias、現在のVersion、新しいVersionを記載します。CodeBuildが影響対象ごとにAppSpecを生成し、CodeDeploy APIからDeploymentを作成するため、非対象関数にはDeploymentを作成しません。

複数関数の途中で失敗した場合は基準SHAを更新しません。次回実行時に同じ差分を再評価し、必要な関数を再度デプロイします。

## インフラ変更

通常の関数コード更新とCDKによる構成更新を区別します。

- アプリケーションコードのみ: 選択ビルド後にLambdaコードを更新
- Lambda設定、Route、IAM、Pipeline変更: CDKの`diff`と`deploy`が必要
- 関数の追加／削除: 先にCDKでリソース構成を反映し、その後コードをデプロイ

初期段階ではインフラ変更の自動`cdk deploy`は行いません。IAMや公開Routeの変更をレビュー後に明示的に反映します。自動化する場合は専用Stage、承認Action、環境別Roleを追加します。

## IAM

CodeBuild Roleに必要な主な権限は次のとおりです。Resourceは対象関数と対象Parameterへ限定します。

- `lambda:UpdateFunctionCode`
- `lambda:GetFunctionConfiguration`
- `lambda:PublishVersion`
- `lambda:GetAlias`
- `codedeploy:CreateDeployment`
- `codedeploy:GetDeployment`
- `ssm:GetParameter`
- `ssm:PutParameter`
- CloudWatch LogsへのBuildログ出力
- Source取得に必要なCodeConnections／S3権限

Lambda実行Roleにはデプロイ権限を付与しません。

## 失敗時の原則

- テスト、ビルド、Version発行、1つでもCodeDeploy Deploymentが失敗したらPipelineを失敗させる。
- 失敗時は基準SHAを更新しない。
- 対象0件は正常終了とし、基準SHAだけ更新する。
- タイムアウトで成否不明の場合はLambdaの`LastUpdateStatus`を確認する。
- Pipeline実行は直列化し、古いCommitが新しいCommitを上書きしないようにする。
