# デプロイ手順

## 前提

- AWS CLIとAWS CDKが利用できる。
- ホストの`~/.aws`に対象環境のProfileがある。
- Dev Containerでは`.devcontainer/.env`の`AWS_PROFILE`が対象Profileを指している。
- GitHubとのCodeConnectionが作成され、利用可能になっている。
- 環境設定にAWS Account、Region、Repository、Branch、Connection ARNが設定されている。

認証情報そのものはリポジトリへ保存しません。

## 初回構築

1. AWS認証を確認する。

   ```bash
   aws sts get-caller-identity
   ```

2. 統合CDK Appを確認する。

   ```bash
   task infra:synth
   task infra:diff
   ```

3. Network StackとStorage Stackをデプロイする。

   ```bash
   task deploy:stack
   ```

4. 3つのECR Repositoryへ、各関数のarm64イメージを同じGit SHAタグでpushする。

   ```bash
   task image:push IMAGE_TAG=<Git SHA>
   ```

5. Application Stackをデプロイする。

   ```bash
   task deploy:application:init IMAGE_TAG=<Git SHA>
   ```

6. CDKが3つのLambda、初期Version、`live` Alias、CodeDeploy、API Gateway Routeを作成したことを確認する。

7. Pipelineを実行し、SSMのSHAを`BASE_SHA`として差分判定することを確認する。

8. API GatewayのEndpointで各Routeを確認する。

   ```bash
   curl "https://<api-id>.execute-api.ap-northeast-1.amazonaws.com/users"
   curl "https://<api-id>.execute-api.ap-northeast-1.amazonaws.com/orders"
   curl "https://<api-id>.execute-api.ap-northeast-1.amazonaws.com/posts"
   ```

## 通常デプロイ

対象Branchへのpushを契機にPipelineが起動します。

1. Source ActionのCommit IDがpushしたCommitと一致する。
2. CodeBuildログに`BASE_SHA`、`HEAD_SHA`、対象関数、判定理由が出力される。
3. 対象関数だけがビルドされる。
4. 対象Lambdaに新しいVersionが発行される。
5. 非対象Lambdaが更新されていない。
6. CodeDeployのAll-at-once Deploymentが成功する。
7. `live` Aliasが新Versionを100%参照する。
8. API Gateway経由のレスポンスが期待どおりである。
9. SSM Parameterが`HEAD_SHA`へ更新される。

## ローカル確認

実装後はTaskfileから同じ処理を呼び出せるようにします。

```bash
task test
task detect BASE_SHA=<sha> HEAD_SHA=<sha>
task infra:synth
task deploy:stack
task deploy:application
```

ローカルのビルド成果物は一時ディレクトリへ生成し、Gitへ追加しません。

## ロールバック

CodeDeploy Deploymentが失敗または停止した場合は、Deployment GroupのAutoRollback設定により旧Versionへ戻せるようにします。ただし、初期構成ではアプリケーション異常を検知するCloudWatch Alarmを必須にしないため、正常終了後の論理不具合までは自動検知できません。

手動ロールバックは旧VersionをTargetVersionとするCodeDeploy Deploymentを作成します。緊急時以外は問題Commitをrevertし、Pipelineから再デプロイしてコードと履歴を一致させます。

より厳密な本番運用ではCloudWatch AlarmとPreTraffic／PostTraffic Hookを追加し、All-at-onceからCanaryまたはLinearへ変更します。

## 関数追加

1. `cmd/<function-name>`とテストを追加する。
2. CDK設定へLambda関数とHTTP API Routeを追加する。
3. `cdk diff`で公開範囲とIAM差分をレビューする。
4. `cdk deploy`で関数とRouteを作成する。
5. Pipelineを実行し、関数コードをデプロイする。

## 関数削除

先にAPI Routeへのアクセス停止と利用状況を確認し、CDK差分をレビューしてから削除します。変更影響判定では`cmd`削除を全関数対象として扱います。
