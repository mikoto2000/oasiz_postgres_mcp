# oasiz-postgres-mcp

Go で実装した PostgreSQL メタデータ参照用 MCP サーバーです。

LLM クライアントから PostgreSQL のスキーマ情報・テーブル情報を安全に参照するためのサーバーで、読み取り専用として動作します。任意 SQL 実行機能は提供しません。

## 機能

以下の MCP Tool を提供します。

- `list_schemas`: PostgreSQL 内のユーザー定義スキーマ一覧を返します。
- `list_tables`: 指定スキーマ内のテーブル一覧を返します。必要に応じて view / materialized view も含められます。
- `get_table_definition`: 指定テーブルの定義情報を DDL 文字列ではなく構造化 JSON として返します。

## 安全性

- 任意 SQL 実行機能はありません。
- `pg_catalog` を参照してメタデータのみを取得します。
- `pg_catalog`, `information_schema`, `pg_toast%`, `pg_temp_%` などのシステム系スキーマは対象外です。
- 接続情報・認証情報・owner はレスポンスに含めません。
- 件数やカラム数、インデックス数が上限を超える場合は `RESULT_TOO_LARGE` を返します。

## 設定

環境変数で設定します。

- `POSTGRES_DSN`: PostgreSQL 接続文字列。必須です。
- `OASIZ_EXCLUDE_SCHEMAS`: 除外するスキーマ名または glob パターンをカンマ区切りで指定します。
- `OASIZ_EXCLUDE_TABLES`: 除外するテーブル名または glob パターンをカンマ区切りで指定します。
- `OASIZ_MAX_TABLES`: `list_tables` で返す最大テーブル数です。デフォルトは `500` です。
- `OASIZ_MAX_COLUMNS`: `get_table_definition` で返す最大カラム数です。デフォルトは `200` です。
- `OASIZ_MAX_INDEXES`: `get_table_definition` で返す最大インデックス数です。デフォルトは `100` です。
- `OASIZ_CONFIG`: JSON 設定ファイルのパスです。指定した場合はファイル設定を読み込みます。

## JSON 設定ファイル例

```json
{
  "postgres_dsn": "postgres://readonly:secret@localhost:5432/app?sslmode=disable",
  "metadata": {
    "exclude_schemas": ["internal_*"],
    "exclude_tables": ["audit_*"],
    "max_tables": 500,
    "max_columns": 200,
    "max_indexes": 100
  }
}
```

## 起動方法

stdio transport で起動します。

```sh
POSTGRES_DSN='postgres://readonly:secret@localhost:5432/app?sslmode=disable' go run ./cmd/oasiz-postgres-mcp
```

JSON 設定ファイルを使う場合:

```sh
OASIZ_CONFIG=./config.json go run ./cmd/oasiz-postgres-mcp
```

MCP クライアントから使う場合は、先にバイナリをビルドしておくと設定が安定します。

```sh
go build -o ./bin/oasiz-postgres-mcp ./cmd/oasiz-postgres-mcp
```

## MCP サーバー設定例

以下の例では、`/path/to/oasiz_postgres_mcp` をこのリポジトリの実際のパスに置き換えてください。`POSTGRES_DSN` には読み取り専用ユーザーの接続文字列を指定することを推奨します。

### Claude Code

プロジェクトで共有する場合は、リポジトリ直下に `.mcp.json` を作成します。

```json
{
  "mcpServers": {
    "oasiz-postgres": {
      "command": "/path/to/oasiz_postgres_mcp/bin/oasiz-postgres-mcp",
      "args": [],
      "env": {
        "POSTGRES_DSN": "postgres://readonly:secret@localhost:5432/app?sslmode=disable",
        "OASIZ_EXCLUDE_SCHEMAS": "internal_*",
        "OASIZ_EXCLUDE_TABLES": "audit_*",
        "OASIZ_MAX_TABLES": "500",
        "OASIZ_MAX_COLUMNS": "200",
        "OASIZ_MAX_INDEXES": "100"
      }
    }
  }
}
```

CLI で追加する場合:

```sh
claude mcp add --transport stdio oasiz-postgres \
  --env POSTGRES_DSN='postgres://readonly:secret@localhost:5432/app?sslmode=disable' \
  --env OASIZ_EXCLUDE_SCHEMAS='internal_*' \
  --env OASIZ_EXCLUDE_TABLES='audit_*' \
  -- /path/to/oasiz_postgres_mcp/bin/oasiz-postgres-mcp
```

### Codex

Codex では `~/.codex/config.toml`、または信頼済みプロジェクトの `.codex/config.toml` に MCP サーバーを設定できます。

```toml
[mcp_servers.oasiz-postgres]
command = "/path/to/oasiz_postgres_mcp/bin/oasiz-postgres-mcp"
startup_timeout_sec = 10
tool_timeout_sec = 60
enabled = true

[mcp_servers.oasiz-postgres.env]
POSTGRES_DSN = "postgres://readonly:secret@localhost:5432/app?sslmode=disable"
OASIZ_EXCLUDE_SCHEMAS = "internal_*"
OASIZ_EXCLUDE_TABLES = "audit_*"
OASIZ_MAX_TABLES = "500"
OASIZ_MAX_COLUMNS = "200"
OASIZ_MAX_INDEXES = "100"
```

CLI で追加する場合:

```sh
codex mcp add oasiz-postgres \
  --env POSTGRES_DSN='postgres://readonly:secret@localhost:5432/app?sslmode=disable' \
  --env OASIZ_EXCLUDE_SCHEMAS='internal_*' \
  --env OASIZ_EXCLUDE_TABLES='audit_*' \
  -- /path/to/oasiz_postgres_mcp/bin/oasiz-postgres-mcp
```

## Tool 一覧

### `list_schemas`

入力はありません。

出力例:

```json
{
  "schemas": [
    {
      "name": "app"
    },
    {
      "name": "public"
    }
  ]
}
```

### `list_tables`

入力例:

```json
{
  "schema": "public",
  "include_views": false
}
```

出力例:

```json
{
  "schema": "public",
  "tables": [
    {
      "name": "orders",
      "type": "BASE TABLE",
      "comment": "注文",
      "column_count": 12
    },
    {
      "name": "users",
      "type": "BASE TABLE",
      "comment": "ユーザーマスタ",
      "column_count": 8
    }
  ]
}
```

### `get_table_definition`

入力例:

```json
{
  "schema": "public",
  "table": "users",
  "include_indexes": true,
  "include_foreign_keys": true,
  "include_comments": true
}
```

出力には以下を含みます。

- スキーマ名
- テーブル名
- テーブル種別
- テーブルコメント
- カラム一覧
- 主キー
- 外部キー
- ユニーク制約
- インデックス
- カラムコメント

## エラーコード

Tool 実行時の業務エラーは構造化レスポンス内の `error.code` として返します。

- `SCHEMA_NOT_FOUND`: 指定されたスキーマが存在しない、または参照対象外です。
- `TABLE_NOT_FOUND`: 指定されたテーブルが存在しない、または参照対象外です。
- `RESULT_TOO_LARGE`: 結果件数、カラム数、またはインデックス数が設定上限を超えています。
