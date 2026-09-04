# ディレクトリ構成

## 目標構成

```text
codepipeline_monorepo_practice/
├── cmd/
│   ├── user/
│   │   └── main.go
│   ├── order/
│   │   └── main.go
│   └── post/
│       └── main.go
├── internal/
│   ├── user/
│   ├── order/
│   ├── post/
│   └── common/
├── build/
│   ├── detect-affected.sh
│   ├── build-lambda.sh
│   └── deploy-lambda.sh
├── infra/
│   ├── application/
│   │   ├── main.go
│   │   ├── config/
│   │   └── stacks/
│   │       ├── api.go
│   │       ├── lambda.go
│   │       ├── deployment.go
│   │       └── pipeline.go
│   └── common/
├── docs/
├── go.mod
├── go.sum
└── Taskfile.yml
```

この構成は目標形です。既存の単一`cmd/main.go`とECS向けCDKは、実装時に段階的に移行します。

## 責務

### `cmd/user`、`cmd/order`、`cmd/post`

3つのLambdaそれぞれのエントリーポイントです。イベントの受け取り、ユースケース呼び出し、API Gateway向けレスポンスへの変換に限定します。

ディレクトリ名を、変更影響判定、Lambda関数名の末尾、ビルド成果物名、デプロイ対象名の基準とします。

### `internal`

アプリケーション固有処理と共有処理を配置します。CI都合でLambdaごとのディレクトリへ複製せず、Goパッケージとして自然な責務で分割します。

### `build`

ローカルとCodeBuildの双方から実行できるスクリプトを配置します。AWS固有の値は引数または環境変数で受け取り、アカウントIDやARNを埋め込みません。

### `infra`

CDK AppとStackを配置します。関数一覧、API Route、メモリ、タイムアウトなど、環境差分を型付き設定として管理します。

## 関数追加時の変更箇所

1. `cmd/<function-name>/main.go`を追加する。
2. 必要な処理を`internal`へ追加する。
3. CDK設定へ関数とAPI Routeの対応を追加する。
4. 関数固有のIAM権限、環境変数、タイムアウトを定義する。
5. 単体テストと変更影響判定のテストケースを追加する。
