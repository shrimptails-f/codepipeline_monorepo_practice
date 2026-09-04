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
