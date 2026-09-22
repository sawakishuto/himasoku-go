.DEFAULT_GOAL := help

APP_NAME       := himasoku
BIN            := bin/$(APP_NAME)
MAIN_PKG       := ./cmd/server
MIGRATIONS_DIR := db/migrations

# ローカル開発用。docker-compose.yml の設定と揃えてある。
# 本番などでは環境変数で上書きする: make migrate-up DATABASE_URL=postgres://...
#
# 接続先は 2 つある。用途を取り違えると RLS が無効になるので注意すること。
#   DATABASE_URL     所有者 himasoku。BYPASSRLS を持つ。マイグレーションと配信ワーカー用
#   APP_DATABASE_URL himasoku_app。RLS が適用される。API のリクエスト処理用
DATABASE_URL     ?= postgres://himasoku:himasoku@localhost:15432/himasoku?sslmode=disable
APP_DATABASE_URL ?= postgres://himasoku_app:himasoku_app@localhost:15432/himasoku?sslmode=disable

.PHONY: help
help: ## このヘルプを表示
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ---------------------------------------------------------------- アプリケーション

.PHONY: run
run: ## サーバーを起動する (http://localhost:8080)
	go run $(MAIN_PKG)

.PHONY: build
build: ## バイナリを bin/ にビルドする
	go build -o $(BIN) $(MAIN_PKG)

.PHONY: clean
clean: ## ビルド成果物を削除する
	rm -rf bin

# ---------------------------------------------------------------- テスト・静的解析

.PHONY: test
test: ## テストを実行する
	go test ./...

.PHONY: test-cover
test-cover: ## カバレッジを計測して HTML で開く
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: lint
lint: ## golangci-lint を実行する
	golangci-lint run

.PHONY: fmt
fmt: ## gofmt でフォーマットする
	gofmt -w ./cmd ./internal

.PHONY: vet
vet: ## go vet を実行する
	go vet ./...

.PHONY: tidy
tidy: ## go.mod / go.sum を整理する
	go mod tidy

.PHONY: check
check: fmt vet lint test ## fmt・vet・lint・test をまとめて実行する

# ---------------------------------------------------------------- DB

.PHONY: db-up
db-up: ## ローカルの PostgreSQL を起動する
	docker compose up -d

.PHONY: db-down
db-down: ## ローカルの PostgreSQL を停止する
	docker compose down

.PHONY: db-reset
db-reset: ## DB をボリュームごと作り直してマイグレーションを流す
	docker compose down -v
	docker compose up -d
	@echo "PostgreSQL の起動を待っています..."
	@until docker compose exec -T postgres pg_isready -U himasoku >/dev/null 2>&1; do sleep 1; done
	$(MAKE) migrate-up
	$(MAKE) db-app-password

.PHONY: db-app-password
db-app-password: ## ローカル用に himasoku_app のパスワードを設定する
	@# マイグレーションはロールを PASSWORD NULL で作る。資格情報をリポジトリに
	@# 残さないため、ローカルの値だけここで与える。本番はシークレット管理側で設定する。
	docker compose exec -T postgres psql -U himasoku -d himasoku -q \
		-c "ALTER ROLE himasoku_app WITH PASSWORD 'himasoku_app';"

.PHONY: psql
psql: ## ローカル DB に psql で接続する (所有者。RLS は適用されない)
	docker compose exec postgres psql -U himasoku -d himasoku

.PHONY: psql-app
psql-app: ## アプリロールで psql 接続する (RLS の挙動を確認したいとき)
	@echo "SET app.current_user_id = '<users.id>'; を実行してから参照すること"
	docker compose exec -e PGPASSWORD=himasoku_app postgres \
		psql -h 127.0.0.1 -U himasoku_app -d himasoku

# ---------------------------------------------------------------- マイグレーション

.PHONY: migrate-up
migrate-up: ## マイグレーションを最新まで適用する
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down: ## マイグレーションを 1 つ巻き戻す
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

.PHONY: migrate-create
migrate-create: ## マイグレーションを新規作成する (make migrate-create name=create_users)
	@test -n "$(name)" || { echo "name= を指定してください  例: make migrate-create name=create_users"; exit 1; }
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

.PHONY: migrate-version
migrate-version: ## 現在のマイグレーションバージョンを表示する
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" version

.PHONY: migrate-force
migrate-force: ## dirty 状態を指定バージョンで強制解除する (make migrate-force version=1)
	@test -n "$(version)" || { echo "version= を指定してください  例: make migrate-force version=1"; exit 1; }
	migrate -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" force $(version)

# ---------------------------------------------------------------- コード生成

.PHONY: sqlc
sqlc: ## sqlc でクエリから Go コードを生成する
	sqlc generate

.PHONY: sqlc-vet
sqlc-vet: ## sqlc のクエリを検査する
	sqlc vet

# ---------------------------------------------------------------- セットアップ

.PHONY: tools
tools: ## 開発に必要な CLI をインストールする
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
