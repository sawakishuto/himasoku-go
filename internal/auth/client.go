package auth

import (
	"context"
)

// Provider は user_identities.provider に対応する。どのプロバイダの
// トークンかを知っているのは検証した実装だけなので、そこで詰める。
const ProviderFirebase = "firebase"

type Token struct {
	// Provider は検証した実装が設定する。Subject と組にして
	// user_identities を引く。sub は発行者の中でしか一意ではない。
	Provider string
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
