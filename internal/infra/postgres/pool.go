package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool は API 用の接続プールを作る。
//
// 接続先は必ず APP_DATABASE_URL（himasoku_app）にすること。所有者の
// DATABASE_URL は BYPASSRLS を持つため、そちらで繋ぐと Row Level Security が
// 一切評価されない。エラーも警告も出ないので、取り違えると静かに無防備になる。
func NewPool(ctx context.Context) (*pgxpool.Pool, error) {
	dsn := os.Getenv("APP_DATABASE_URL")
	if dsn == "" {
		return nil, errors.New("APP_DATABASE_URL is not set")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// 設定ミスを起動時に気づけるようにする。遅延接続のままだと
	// 最初のリクエストまで誤りが表面化しない。
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to reach database: %w", err)
	}

	return pool, nil
}
