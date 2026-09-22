# Himasoku (Go)

「今ヒマ」をグループ内で共有し、プッシュ通知で誘い合うアプリのバックエンド。

既存の Rails 実装（[`sawakishuto/HimasokuBackend`](https://github.com/sawakishuto/HimasokuBackend)）
の仕様を引き継ぎつつ、スキーマと認証まわりを設計し直している。
設計の意図と各ユースケースの詳細は [`docs/`](docs/README.md) にある。

## 技術スタック

| 領域 | 採用 |
|------|------|
| 言語 | Go 1.26 |
| HTTP | 標準ライブラリの `net/http`（Go 1.22 以降のパターンルーティング） |
| DB | PostgreSQL 17 + [pgx/v5](https://github.com/jackc/pgx) |
| クエリ | [sqlc](https://sqlc.dev) で SQL から Go を生成 |
| マイグレーション | [golang-migrate](https://github.com/golang-migrate/migrate) |
| 認証 | Firebase Authentication（ID トークンの検証のみ） |
| 通知 | APNs |

## セットアップ

### 必要なもの

- Go 1.26 以降
- Docker（ローカルの PostgreSQL 用）
- Firebase プロジェクトのサービスアカウント鍵（トークン検証に使う）

### 手順

```bash
make tools      # migrate / sqlc / golangci-lint を入れる
make db-reset   # PostgreSQL を起動し、マイグレーションとロール設定まで済ませる
make run        # http://localhost:8080
```

疎通確認。

```bash
curl -i http://localhost:8080/health   # 200
curl -i http://localhost:8080/groups   # 401（トークンが無いため）
```

### Firebase の資格情報

Admin SDK は Application Default Credentials を読む。鍵ファイルは
**リポジトリの外**に置き、パスを環境変数で渡す。

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
```

資格情報が無くてもサーバーは起動する。失敗するのは実際にトークンを
検証する時点なので、`/health` だけを見て設定できたと判断しないこと。

## 環境変数

| 変数 | 既定値（ローカル） | 用途 |
|------|------------------|------|
| `APP_DATABASE_URL` | `himasoku_app` で接続 | **API 本体**。Row Level Security が適用される |
| `DATABASE_URL` | `himasoku` で接続 | マイグレーションと配信ワーカー。`BYPASSRLS` を持つ |
| `GOOGLE_APPLICATION_CREDENTIALS` | なし | Firebase サービスアカウント鍵のパス |

接続先が 2 つあるのは意図的で、**取り違えると静かに無防備になる**。
所有者ロールは `BYPASSRLS` を持つため、そちらで繋ぐとポリシーが一切
評価されない。エラーも警告も出ない。API は必ず `APP_DATABASE_URL` を使う。

ローカルの既定値は `Makefile` に書いてあるので、`make run` はそのまま動く。

## エンドポイント

`GET /health` 以外はすべて認証が必要。

```
Authorization: Bearer <Firebase ID Token>
```

トークンが有効であることと、`users` に行があることは別の条件として扱う。
Google のサインインは済んだがプロフィール未登録、という状態が正常に
存在するため、エンドポイントは要求するものによって 3 層に分かれる。

| 層 | 対象 | トークンが無効 | 未登録 |
|----|------|--------------|--------|
| 認証不要 | `GET /health` | — | — |
| 認証のみ | `POST /users`, `GET /me` | `401` | 通す |
| 認証 + 登録済み | それ以外すべて | `401` | `403 registration_required` |

登録エンドポイントを 3 層目に置くと、登録するために登録済みである必要が
生じて誰も登録できなくなる。2 層目があるのはそのため。

### 実装状況

| エンドポイント | 状態 |
|--------------|------|
| `GET /health` | 実装済み |
| `POST /users` | 実装済み（[UC-01](docs/usecases/onboarding.md#uc-01-ユーザー登録)） |
| それ以外（UC-02 〜 UC-15） | 設計のみ。[`docs/`](docs/README.md) を参照 |

## ディレクトリ構成

```
cmd/server/          エントリポイント
db/
  migrations/        golang-migrate の SQL
  queries/           sqlc の入力
docs/                設計ドキュメント（ユースケース・スキーマ）
internal/
  auth/              トークン検証。Client インターフェースと Firebase 実装
  dbgen/             sqlc の生成物。手で編集しない
  domain/user/       entity / repository（インターフェース）/ usecase
  handler/           HTTP ハンドラ
    middleware/      認証ミドルウェア
  infra/postgres/    repository の実装と接続プール
  server/            ルーティングと起動・停止
```

ドメイン層はインターフェースだけを持ち、`infra/postgres` がそれを実装する。
`middleware` が `UserResolver` を自前で定義しているのも同じ理由で、
利用する側がインターフェースを持つことで保存先を知らずに済ませている。

## 開発

```bash
make help    # ターゲット一覧
make check   # fmt・vet・lint・test をまとめて
```

### スキーマを変えるとき

マイグレーションとクエリの生成物がずれると、ビルドは通るのに実行時に
落ちる。次の順で回すこと。

```bash
make migrate-create name=add_something   # db/migrations に 2 ファイル作る
make migrate-up                          # 適用する
make sqlc                                # internal/dbgen を作り直す
```

`down` も必ず書いて、`make migrate-down && make migrate-up` で往復すること。

### RLS の挙動を確かめたいとき

```bash
make psql-app   # アプリロールで接続（RLS が効く）
make psql       # 所有者で接続（RLS を素通りする）
```

`psql-app` では実行主体を自分で指定する。

```sql
SET app.current_user_id = '<users.id>';
```

## ドキュメント

| 内容 | 場所 |
|------|------|
| ユースケース一覧・用語・前提 | [docs/README.md](docs/README.md) |
| テーブル設計と RLS ポリシー | [docs/schema.md](docs/schema.md) |
| 登録とデバイストークン | [docs/usecases/onboarding.md](docs/usecases/onboarding.md) |
| グループ | [docs/usecases/groups.md](docs/usecases/groups.md) |
| 暇の共有 | [docs/usecases/availability-sharing.md](docs/usecases/availability-sharing.md) |
| 通知配信 | [docs/usecases/notification-delivery.md](docs/usecases/notification-delivery.md) |
