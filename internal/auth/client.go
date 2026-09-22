package auth

import (
	"context"
)

type Token struct {
	AuthTime int64
	Issuer   string
	Audience string
	Expires  int64
	IssuedAt int64
	Subject  string
	UID      string
	Claims   map[string]interface{}
}

type Client interface {
	VerifyToken(ctx context.Context, rawToken string) (*Token, error)
}

func NewClient() (Client, error) {
	return NewFirebaseAuthClient()
}
