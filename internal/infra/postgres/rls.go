package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sawakishuto/himasoku-go/internal/dbgen"
)

// runAsUser はトランザクション内で app.current_user_id を設定して fn を実行する。
//
// devices など RLS 対象の操作は、プールから取った接続だけではポリシーが
// 評価されない。SET LOCAL はトランザクション終了で巻き戻るので、接続プール
// に利用者 ID が残らない。
func runAsUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, fn func(context.Context, *dbgen.Queries) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // Commit 成功時は no-op

	// set_config(..., true) は SET LOCAL と同じ。第 3 引数 true が is_local。
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_user_id', $1, true)", userID.String()); err != nil {
		return fmt.Errorf("set app.current_user_id: %w", err)
	}

	if err := fn(ctx, dbgen.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
