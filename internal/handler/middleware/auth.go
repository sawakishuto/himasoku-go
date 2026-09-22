package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/sawakishuto/himasoku-go/internal/auth"
)

var errMissingBearer = errors.New("authorization header is missing or malformed")

type Authenticator struct {
	client auth.Client
}

func NewAuthenticator(client auth.Client) *Authenticator {
	return &Authenticator{client: client}
}

// Require は ID トークンを検証し、確定したトークンを context に載せて次へ渡す。
func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := a.verify(r)
		if err != nil {
			unauthorized(w)
			return
		}
		next.ServeHTTP(w, r.WithContext(withToken(r.Context(), token)))
	})
}

func (a *Authenticator) verify(r *http.Request) (*auth.Token, error) {
	// 検証側は JWT 本体を期待する。"Bearer " が付いたままだと必ず失敗する。
	raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || raw == "" {
		return nil, errMissingBearer
	}
	return a.client.VerifyToken(r.Context(), raw)
}

// 失敗理由は返さない。総当たりの手掛かりになる。
func unauthorized(w http.ResponseWriter) {
	http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
}

type ctxKey struct{}

var tokenKey ctxKey

func withToken(ctx context.Context, token *auth.Token) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

func TokenFrom(ctx context.Context) (*auth.Token, bool) {
	token, ok := ctx.Value(tokenKey).(*auth.Token)
	return token, ok
}
