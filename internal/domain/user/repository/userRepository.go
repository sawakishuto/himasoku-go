package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sawakishuto/himasoku-go/internal/domain/user/entity"
)

// ErrUserNotFound は該当する利用者が存在しないことを表す。
//
// 実装側の事情（pgx.ErrNoRows など）をそのまま返すと、呼び出し側が
// 判定のために DB ライブラリを import することになり、
// インターフェースで切った意味が無くなる。
var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
}
