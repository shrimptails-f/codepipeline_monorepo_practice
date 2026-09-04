# トラブルシューティング

## AWS認証情報を取得できない

```text
Unable to locate credentials
The config profile (...) could not be found
```

- ホストの`~/.aws/config`と`~/.aws/credentials`にProfileが存在するか。
- `.devcontainer/.env`の`AWS_PROFILE`がProfile名と一致するか。
- Dev Containerを設定変更後にリビルドしたか。
- SSOの場合、ホスト側で`aws sso login --profile <profile>`を実行済みか。
- コンテナ内の`/home/dev/.aws`が読み取り専用でマウントされているか。

## 変更対象が常に全関数になる

CodeBuildログの判定理由を確認します。基準SHA取得失敗、`go.mod`／`go.sum`変更、共有Goファイルの削除／移動、判定スクリプトやインフラの変更は、仕様として全関数対象です。

SSM Parameterが存在しない初回実行は例外で、全関数を対象にせず、現在のCommit SHAを初期基準として登録します。

Clone形式のSource ArtifactとGit履歴のfetch設定も確認します。

## 変更した関数が対象にならない

1. `BASE_SHA..HEAD_SHA`に対象ファイルが含まれるか。
2. 変更ディレクトリを`go list`でImport Pathへ変換できるか。
3. `go list -deps ./cmd/user`、`./cmd/order`、`./cmd/post`の該当結果に変更Packageが含まれるか。
4. Build ConstraintやOS／Architecture条件によって依存関係が変わっていないか。
5. 削除／移動が保守的な全対象ルールへ入っているか。

判定に確信が持てない場合は全関数デプロイへフォールバックします。

## Lambda更新が失敗する

```bash
aws lambda get-function-configuration \
  --function-name "<env>-codepipeline-monorepo-practice-<function-name>" \
  --query '{State:State,LastUpdateStatus:LastUpdateStatus,Reason:LastUpdateStatusReason}'
```

- ZIP直下に`bootstrap`があるか。
- `bootstrap`が`linux/arm64`向けにビルドされているか。
- Lambda Runtimeが`provided.al2023`か。
- CodeBuild Roleに対象関数の更新権限があるか。
- ZIPサイズが上限を超えていないか。
- 同じ関数を別Pipelineが同時更新していないか。

## API Gatewayが5xxを返す

1. API GatewayのRouteとIntegration先Lambdaを確認する。
2. LambdaのCloudWatch Logsでpanicやタイムアウトを確認する。
3. API Gateway HTTP APIのイベント形式にHandlerが対応しているか確認する。
4. Lambda Resource PolicyでAPI GatewayからのInvokeが許可されているか確認する。
5. Handlerが適切なStatus Code、Header、Bodyを返しているか確認する。

## CodeDeploy Deploymentが失敗する

- AppSpecの関数名、Alias、CurrentVersion、TargetVersionが実在するか。
- `live` AliasがCurrentVersionを参照しているか。
- TargetVersionが発行済みの数値Versionか。`$LATEST`は指定しない。
- CodeDeploy Service RoleにAliasの更新権限があるか。
- Deploymentを途中停止した場合、Aliasに意図しないWeightが残っていないか。
- HookまたはCloudWatch Alarmを設定した場合、その失敗理由を確認する。

## API Gatewayが404を返す

- HTTP MethodとRoute PathがCDK定義に一致するか。
- `$default` Stageまたは指定Stageへデプロイされているか。
- URLに不要なStage名を付けていないか。
- 関数追加時にCDKをデプロイしたか。

## Pipeline成功後も古い応答になる

- PipelineのCommit IDと期待するCommitが一致するか。
- 変更判定で対象関数に含まれたか。
- Lambdaに新Versionが発行され、`live` AliasがそのVersionを参照しているか。
- 対象関数のCodeDeploy Deploymentが成功しているか。
- API Routeが別環境／別関数を参照していないか。
- API Gateway、CloudFront、クライアント側のCacheがないか。

## 基準SHAを更新してはいけないケース

テスト、ビルド、Version発行、CodeDeploy Deployment、Alias確認のどれかが失敗した場合、SSM Parameterを更新しません。手動でSHAを進めると未デプロイ変更が次回差分から落ちるため、障害調査中も原則として書き換えません。
