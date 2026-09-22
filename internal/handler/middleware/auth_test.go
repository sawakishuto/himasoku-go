package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/sawakishuto/himasoku-go/internal/auth"
)

type fakeVerifier struct {
	token *auth.Token
	err   error
}

func (f fakeVerifier) VerifyToken(context.Context, string) (*auth.Token, error) {
	return f.token, f.err
}

type fakeResolver struct {
	id    uuid.UUID
	found bool
	err   error
}

func (f fakeResolver) ResolveUserID(context.Context, string, string) (uuid.UUID, bool, error) {
	return f.id, f.found, f.err
}

// 有効なトークンを返す検証器。resolver 側の分岐を見たいときに使う。
func validToken() fakeVerifier {
	return fakeVerifier{token: &auth.Token{Provider: auth.ProviderFirebase, Subject: "sub-1"}}
}

func authorized(r *http.Request) *http.Request {
	r.Header.Set("Authorization", "Bearer eyJhbGciOi")
	return r
}

// "Bearer " を剥がしてから検証器へ渡していること。
// 付いたままだと JWT として解釈できず、全リクエストが 401 になる。
func TestVerifyStripsBearerPrefix(t *testing.T) {
	var got string
	a := NewAuthenticator(recordingVerifier{raw: &got}, nil)

	r := authorized(httptest.NewRequest(http.MethodGet, "/me", nil))
	if _, err := a.verify(r); err != nil {
		t.Fatalf("予期しないエラー: %v", err)
	}

	if got != "eyJhbGciOi" {
		t.Errorf("検証器が受け取った値 = %q, 期待値 = %q", got, "eyJhbGciOi")
	}
}

type recordingVerifier struct{ raw *string }

func (v recordingVerifier) VerifyToken(_ context.Context, raw string) (*auth.Token, error) {
	*v.raw = raw
	return &auth.Token{Provider: auth.ProviderFirebase, Subject: "sub-1"}, nil
}

// 登録済みなら users.id が context に載ること。
func TestRequireAttachesUserID(t *testing.T) {
	want := uuid.New()
	a := NewAuthenticator(validToken(), fakeResolver{id: want, found: true})

	var got uuid.UUID
	var ok bool
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, ok = UserIDFrom(r.Context())
	})

	rec := httptest.NewRecorder()
	a.Require(next).ServeHTTP(rec, authorized(httptest.NewRequest(http.MethodGet, "/me", nil)))

	if !ok {
		t.Fatal("context から users.id を取り出せなかった")
	}
	if got != want {
		t.Errorf("users.id = %v, 期待値 = %v", got, want)
	}
}

// 未登録でも Require は通す。登録用のエンドポイントに到達させる必要がある。
func TestRequireAllowsUnregistered(t *testing.T) {
	a := NewAuthenticator(validToken(), fakeResolver{found: false})

	called := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		if _, ok := UserIDFrom(r.Context()); ok {
			t.Error("未登録なのに users.id が載っている")
		}
		if _, ok := TokenFrom(r.Context()); !ok {
			t.Error("トークンが context に載っていない")
		}
	})

	rec := httptest.NewRecorder()
	a.Require(next).ServeHTTP(rec, authorized(httptest.NewRequest(http.MethodPost, "/users", nil)))

	if !called {
		t.Errorf("未登録のリクエストが弾かれた (ステータス = %d)", rec.Code)
	}
}

// 引き当てに失敗したときは 500。未登録と混同しない。
func TestRequireFailsOnResolverError(t *testing.T) {
	a := NewAuthenticator(validToken(), fakeResolver{err: errors.New("db down")})

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	rec := httptest.NewRecorder()
	a.Require(next).ServeHTTP(rec, authorized(httptest.NewRequest(http.MethodGet, "/me", nil)))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("ステータス = %d, 期待値 = %d", rec.Code, http.StatusInternalServerError)
	}
	if called {
		t.Error("引き当てに失敗したのに次のハンドラが呼ばれた")
	}
}

// 未登録は 403。トークンは有効なので 401 にはしない。
func TestRequireRegisteredRejectsUnregistered(t *testing.T) {
	a := NewAuthenticator(validToken(), fakeResolver{found: false})

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	rec := httptest.NewRecorder()
	a.RequireRegistered(next).ServeHTTP(rec, authorized(httptest.NewRequest(http.MethodGet, "/groups", nil)))

	if rec.Code != http.StatusForbidden {
		t.Errorf("ステータス = %d, 期待値 = %d", rec.Code, http.StatusForbidden)
	}
	if called {
		t.Error("未登録なのに次のハンドラが呼ばれた")
	}
}

// 登録済みなら RequireRegistered を通過すること。
func TestRequireRegisteredAllowsRegistered(t *testing.T) {
	a := NewAuthenticator(validToken(), fakeResolver{id: uuid.New(), found: true})

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })

	rec := httptest.NewRecorder()
	a.RequireRegistered(next).ServeHTTP(rec, authorized(httptest.NewRequest(http.MethodGet, "/groups", nil)))

	if !called {
		t.Errorf("登録済みのリクエストが弾かれた (ステータス = %d)", rec.Code)
	}
}
