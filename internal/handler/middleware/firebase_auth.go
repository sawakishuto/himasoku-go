package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"
)

var errMissingBearer = errors.New("authorization header is missing or malformed")

type Authenticator struct {
	client *firebaseauth.Client
}

func NewAuthenticator(client *firebaseauth.Client) *Authenticator {
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

func (a *Authenticator) verify(r *http.Request) (*firebaseauth.Token, error) {
	// VerifyIDToken は JWT 本体を期待する。"Bearer " が付いたままだと失敗する。
	raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || raw == "" {
		return nil, errMissingBearer
	}
	return a.client.VerifyIDToken(r.Context(), raw)
}

// 失敗理由は返さない。総当たりの手掛かりになる。
func unauthorized(w http.ResponseWriter) {
	http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
}

type ctxKey struct{}

var tokenKey ctxKey

func withToken(ctx context.Context, token *firebaseauth.Token) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

func TokenFrom(ctx context.Context) (*firebaseauth.Token, bool) {
	token, ok := ctx.Value(tokenKey).(*firebaseauth.Token)
	return token, ok
}
