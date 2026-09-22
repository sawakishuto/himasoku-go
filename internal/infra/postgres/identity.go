package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sawakishuto/himasoku-go/internal/dbgen"
)

type IdentityRepository struct {
	queries *dbgen.Queries
}

func NewIdentityRepository(pool *pgxpool.Pool) *IdentityRepository {
	return &IdentityRepository{queries: dbgen.New(pool)}
}

// ResolveUserID は (provider, subject) から users.id を引く。
//
// 未登録は異常ではない。Google のサインインは済ませたがプロフィール登録を
// していない状態が正常に存在するため、見つからないことを error ではなく
// 2 つめの戻り値で表す。
func (r *IdentityRepository) ResolveUserID(ctx context.Context, provider, subject string) (uuid.UUID, bool, error) {
	id, err := r.queries.ResolveUserIDByIdentity(ctx, dbgen.ResolveUserIDByIdentityParams{
		Provider: provider,
		Subject:  subject,
	})
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("failed to resolve user by identity: %w", err)
	}
	if !id.Valid {
		return uuid.Nil, false, nil
	}
	return uuid.UUID(id.Bytes), true, nil
}
