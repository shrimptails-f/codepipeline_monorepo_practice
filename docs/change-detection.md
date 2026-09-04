# Lambda変更影響判定

## 目的

Git差分とGoの依存関係を使い、変更の影響を受ける`cmd/user`、`cmd/order`、`cmd/post`だけをビルド・デプロイします。判定不能な変更は安全側に倒し、3関数すべてを対象とします。

## 基準コミット

SSM Parameter Storeの次のParameterへ、最後に正常処理されたコミットSHAを保存します。

```text
/<env>/codepipeline-monorepo-practice/last-successful-commit
```

Build開始時にParameterのSHAを`BASE_SHA`、SourceステージのSHAを`HEAD_SHA`として差分を取得します。

```bash
git diff --name-status "$BASE_SHA" "$HEAD_SHA"
```

- Parameterは初回Applicationデプロイ前にGit SHAで初期化済みとする。存在しない場合や未設定の場合はPipelineを失敗させる。
- `BASE_SHA`を取得できない場合は必要な履歴をfetchする。
- fetch後も基準コミットが見つからなければ全関数を対象にする。
- Parameterはテストと全対象の更新が成功した後だけ`HEAD_SHA`へ更新する。
- Pipelineの同時実行を直列化し、Parameter更新の競合を防ぐ。

## 判定規則

| 変更 | 対象 |
| --- | --- |
| `cmd/<name>/**` | `<name>`のみ |
| `internal/**`の既存Goパッケージ | そのパッケージへ依存するすべての`cmd` |
| `go.mod`、`go.sum` | 全関数 |
| ビルド／判定スクリプト | 全関数 |
| Lambda、API、IAMに関係するCDKコード | 全関数をビルドし、CDK反映も必要 |
| `docs/**`のみ | デプロイなし |
| テストファイルのみ | テストは実行するが、関数コードに影響しなければデプロイなし |
| 削除、移動、判定不能な共有ファイル | 全関数 |

## Go依存グラフ

各関数が依存するパッケージを取得します。

```bash
go list -deps -f '{{.ImportPath}}' ./cmd/user
go list -deps -f '{{.ImportPath}}' ./cmd/order
go list -deps -f '{{.ImportPath}}' ./cmd/post
```

変更された`.go`ファイルのディレクトリをImport Pathへ変換し、各関数の依存パッケージ集合と照合します。

```text
変更package ∩ cmdの依存package != 空集合
    -> そのcmdを対象にする
```

## 影響範囲検出の精度と懸念点

影響範囲は、関数やメソッド単位ではなくGoパッケージ単位で判定します。変更された関数が実際にLambdaから呼び出されるかまでは解析しません。

例えば、`internal/common`に`FunctionA`と`FunctionB`があり、Lambdaが`FunctionA`だけを使用している場合でも、同じ`internal/common`パッケージの`FunctionB`を変更すると、そのLambdaはデプロイ対象になります。

```text
cmd/user -> internal/common.FunctionAを使用
変更    -> internal/common.FunctionB
判定    -> userをデプロイ
```

`go list -deps`から分かるのは`cmd/user`が`internal/common`パッケージへ直接または間接的に依存していることまでで、パッケージ内のどの関数を使用しているかではありません。このため、変更の見逃しを避ける代わりに、実際には影響を受けないLambdaも対象になる場合があります。

主な懸念点は次のとおりです。

- 共通パッケージが大きくなるほど、その一部だけの変更でも多くのLambdaが対象になりやすい。
- DIコンテナなどで全サービスのProviderを共通パッケージから登録すると、実際に解決・使用する型が一部でも、Goのimport依存上は全サービスへ依存しているように見える場合がある。
- 実行時の設定値、Reflection、外部API、Database Schemaなど、Goのimportグラフに現れない依存関係は`go list -deps`だけでは判定できない。
- Build Tagsや`GOOS`、`GOARCH`、`CGO_ENABLED`が判定時と実際のビルド時で異なると、異なる依存グラフが生成される可能性がある。
- 削除または移動されたパッケージは現在のworktreeに存在しないため、現在側の依存グラフだけでは変更前の影響範囲を特定できない。

現行実装は、精密な関数単位の絞り込みよりも変更の見逃しを避けることを優先します。依存関係を判定できない変更や、削除・移動などの不確実な変更は全関数を対象にします。そのため、将来コードと共通パッケージが複雑になるほど、誤ってデプロイ対象から外す可能性よりも、必要以上に多くのLambdaをデプロイする可能性が高くなります。

影響範囲を保ちやすくするため、共通パッケージを責務ごとに小さく分割し、各LambdaのComposition RootではそのLambdaが必要とするProviderだけを登録します。また、判定時のBuild Tags、`GOOS`、`GOARCH`、`CGO_ENABLED`は実際のLambdaビルドと一致させます。

## 削除と移動

現在のworktreeだけで依存関係を調べると、削除済みパッケージへの旧依存を確認できません。初期実装では誤ったスキップを避けるため、次の変更は全関数を対象とします。

- `internal`配下のGoファイル削除
- Goパッケージのディレクトリ移動
- `cmd`ディレクトリの削除または改名

将来最適化する場合は、Git worktreeを使って`BASE_SHA`と`HEAD_SHA`の両方で依存グラフを生成し、その和集合で判定します。

## 出力契約

判定スクリプトは機械可読なJSONを出力します。

```json
{
  "baseSha": "<BASE_SHA>",
  "headSha": "<HEAD_SHA>",
  "deployAll": false,
  "functions": ["order", "post", "user"],
  "reason": "internal dependency changed"
}
```

対象が0件の場合は成功終了し、Lambdaを更新しません。docsだけの変更を毎回再検出しないよう、すべての検証に成功したら基準SHAは更新します。

初回Applicationデプロイでは、3関数のイメージタグと差分基準SHAを同じGit SHAでSSMへ設定します。Pipelineはこれらが初期化済みであることを前提とします。

## 必須テストケース

- 1つの`cmd`だけの変更
- 1つの関数だけが利用する`internal`変更
- 複数関数が利用する`internal`変更
- `go.mod`または`go.sum`変更
- docsのみの変更
- 共有Goファイルの削除／移動
- Parameterが存在しない、または`UNSET`の場合の異常終了
- Parameterは存在するが`BASE_SHA`をGit履歴から取得できない場合（全関数）
- 複数コミットをまとめてデプロイする場合
