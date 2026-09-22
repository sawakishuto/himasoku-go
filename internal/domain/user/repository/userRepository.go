package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sawakishuto/himasoku-go/internal/domain/user/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
}
