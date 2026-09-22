package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/sawakishuto/himasoku-go/internal/auth"
)

var errMissingBearer = errors.New("authorization header is missing or malformed")

// UserResolver は検証済みトークンの主体から users.id を引く。
// 使う側がインターフェースを持つことで、この層は保存先を知らずに済む。
type UserResolver interface {
	ResolveUserID(ctx context.Context, provider, subject string) (uuid.UUID, bool, error)
}

type Authenticator struct {
	client   auth.Client
	resolver UserResolver
}

func NewAuthenticator(client auth.Client, resolver UserResolver) *Authenticator {
	return &Authenticator{client: client, resolver: resolver}
}

// Require は ID トークンを検証し、users を引いて context に載せる。
//
// 引くだけで、無くてもエラーにしない。認証は本人確認までを担当し、
// 「登録が済んでいるか」をどう扱うかはルート側の判断に委ねる。
// GET /me と POST /users はこの層だけを通す。
func (a *Authenticator) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := a.verify(r)
		if err != nil {
			unauthorized(w)
			return
		}

		ctx := withToken(r.Context(), token)

		userID, found, err := a.resolver.ResolveUserID(ctx, token.Provider, token.Subject)
		if err != nil {
			// 引き当ての失敗は未登録とは別物。認証情報の問題ではないので 500 を返す。
			log.Printf("failed to resolve user: %v", err)
			http.Error(w, `{"error":"Internal Server Error"}`, http.StatusInternalServerError)
			return
		}
		if found {
			ctx = withUserID(ctx, userID)
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireRegistered は Require に加えて users の行が存在することを要求する。
// 登録用のエンドポイント以外はこちらを通す。
func (a *Authenticator) RequireRegistered(next http.Handler) http.Handler {
	return a.Require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserIDFrom(r.Context()); !ok {

			// 401 にしない。トークンは有効で、足りないのは登録だけ。
			// 再認証を促しても解決せず、クライアントは無限に往復する。
			http.Error(w, `{"error":"registration_required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
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

type tokenCtxKey struct{}

type userIDCtxKey struct{}

func withToken(ctx context.Context, token *auth.Token) context.Context {
	return context.WithValue(ctx, tokenCtxKey{}, token)
}

func TokenFrom(ctx context.Context) (*auth.Token, bool) {
	token, ok := ctx.Value(tokenCtxKey{}).(*auth.Token)
	return token, ok
}

func withUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDCtxKey{}, id)
}

// UserIDFrom はハンドラが current_user を取り出すための入口。
// RequireRegistered を通っていれば必ず存在する。
func UserIDFrom(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDCtxKey{}).(uuid.UUID)
	return id, ok
}
