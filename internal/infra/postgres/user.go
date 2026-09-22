package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sawakishuto/himasoku-go/internal/dbgen"
	"github.com/sawakishuto/himasoku-go/internal/domain/user/entity"
	"github.com/sawakishuto/himasoku-go/internal/domain/user/repository"
)

// インターフェースを満たしているかをコンパイル時に検査する。
// メソッドの追加漏れや型の食い違いが、利用箇所ではなくここで止まる。
var _ repository.UserRepository = (*UserRepository)(nil)

type UserRepository struct {
	queries *dbgen.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{queries: dbgen.New(pool)}
}

// Create は User 集約を保存する。
//
// users へ直接 INSERT せず register_user() を呼ぶのは、users と
// user_identities を 1 トランザクションで作るため。片方だけ入ると、
// 本人でも二度と辿り着けない行が残る。
func (r *UserRepository) Create(ctx context.Context, user *entity.User) error {
	id, err := r.queries.RegisterUser(ctx, dbgen.RegisterUserParams{
		// uuid.UUID も pgtype.UUID.Bytes も [16]byte なので、そのまま渡せる。
		UserID:      pgtype.UUID{Bytes: user.ID, Valid: true},
		Provider:    user.Identity.Provider,
		Subject:     user.Identity.Subject,
		DisplayName: user.DisplayName,
		Email:       nullableEmail(user.Email),
	})
	if err != nil {
		// 原因を包んで返す。潰すと一意制約違反と接続断を呼び出し側で区別できない。
		return fmt.Errorf("failed to register user %s: %w", user.ID, err)
	}

	// 先に同じ sub で登録されていれば既存の行の ID が返る。手元の ID は
	// どこにも存在しないので、DB が正とした値に揃えておく。
	user.ID = uuid.UUID(id.Bytes)
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	row, err := r.queries.GetUserByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return nil, wrapNotFound(err, fmt.Sprintf("failed to get user %s", id))
	}
	return toUserEntity(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, wrapNotFound(err, "failed to get user by email")
	}
	return toUserEntity(row), nil
}

func toUserEntity(row dbgen.User) *entity.User {
	user := &entity.User{
		ID:          uuid.UUID(row.ID.Bytes),
		DisplayName: row.DisplayName,
	}
	// email は NULL 可。entity 側は string なので、NULL は空文字として扱う。
	if row.Email != nil {
		user.Email = *row.Email
	}
	return user
}

// wrapNotFound は「見つからない」だけをドメインの語彙に翻訳する。
// それ以外は原因を保ったまま返す。
func wrapNotFound(err error, msg string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", msg, repository.ErrUserNotFound)
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// email は NULL 可かつ UNIQUE。空文字のまま入れると、メールを持たない
// 利用者が 2 人目から一意制約に当たる。NULL は重複とみなされない。
func nullableEmail(email string) *string {
	if email == "" {
		return nil
	}
	return &email
}
