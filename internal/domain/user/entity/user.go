package entity

import (
	"errors"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
}

func NewUser(email string, displayName string) (*User, error) {
	user := &User{
		ID:          uuid.New(),
		Email:       email,
		DisplayName: displayName,
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}
	return user, nil
}

func (u *User) Validate() error {
	if u.Email == "" {
		return errors.New("email is required")
	}
	if u.DisplayName == "" {
		return errors.New("display name is required")
	}
	return nil
}
