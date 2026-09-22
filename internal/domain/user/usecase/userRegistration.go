package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/sawakishuto/himasoku-go/internal/domain/user/entity"
	"github.com/sawakishuto/himasoku-go/internal/domain/user/repository"
)

// ErrInvalidInput は入力が業務上の条件を満たさないことを表す。
// 保存の失敗と区別できないと、利用者の入力ミスまで 500 になる。
var ErrInvalidInput = errors.New("invalid registration input")

type UserRegistration interface {
	Execute(ctx context.Context, input *RegisterUserInput) error
}

type RegisterUserInput struct {
	// Identity は検証済みトークンから来る。リクエスト本文から受け取ると、
	// 他人の sub を名乗って登録できてしまう。
	Identity    entity.Identity
	Email       string
	DisplayName string
}

type userRegistration struct {
	userRepository repository.UserRepository
}

var _ UserRegistration = (*userRegistration)(nil)

func NewUserRegistration(userRepository repository.UserRepository) UserRegistration {
	return &userRegistration{userRepository: userRepository}
}

func (ur *userRegistration) Execute(ctx context.Context, input *RegisterUserInput) error {
	user, err := entity.NewUser(input.Identity, input.Email, input.DisplayName)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidInput, err)
	}

	if err := ur.userRepository.Create(ctx, user); err != nil {
		return fmt.Errorf("failed to register user: %w", err)
	}
	return nil
}
