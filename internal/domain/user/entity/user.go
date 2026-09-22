package entity

import (
	"errors"

	"github.com/google/uuid"
)

// Identity は外部プロバイダ上での主体を表す。
//
// (Provider, Subject) の組で一意になる。sub は発行者の中でしか一意では
// ないので、Provider を欠くと別プロバイダの利用者と衝突しうる。
type Identity struct {
	Provider string
	Subject  string
}

// User は利用者の集約の根。Identity を内側に持つ。
//
// 両者を別々に保存できる形にすると、Identity の無い User が作れてしまう。
// その行は resolve_user_by_identity が引けないため、本人であっても
// 二度と辿り着けない。常に組で作られることを型で保証する。
type User struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	Identity    Identity
}

func NewUser(identity Identity, email string, displayName string) (*User, error) {
	user := &User{
		ID:          uuid.New(),
		Email:       email,
		DisplayName: displayName,
		Identity:    identity,
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}
	return user, nil
}

func (u *User) Validate() error {
	if u.Identity.Provider == "" || u.Identity.Subject == "" {
		return errors.New("identity is required")
	}
	if u.DisplayName == "" {
		return errors.New("display name is required")
	}
	// email は必須にしない。Apple の非公開リレーのように、プロバイダが
	// メールを渡さないサインインが正常に存在する。ここで弾くと、
	// その経路の利用者は登録自体ができなくなる。
	return nil
}
