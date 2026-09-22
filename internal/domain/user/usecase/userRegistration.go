package usecase

import (
	"context"

	"github.com/sawakishuto/himasoku-go/internal/domain/user/repository"
)


type UserRegistration interface {
	Execute(ctx context.Context, input *RegisterUserInput) (*RegisterUserOutput, error)
}

type RegisterUserInput struct {
}

type RegisterUserOutput struct {
}

type userRegistration struct {
	userRepository repository.UserRepository
}

var _ UserRegistration = (*userRegistration)(nil)


func NewUserRegistration(userRepository repository.UserRepository) UserRegistration {
	return &userRegistration{userRepository: userRepository}
}

func (ur *userRegistration) Execute(ctx context.Context, input *RegisterUserInput) (*RegisterUserOutput, error) {

	return nil, nil
}
