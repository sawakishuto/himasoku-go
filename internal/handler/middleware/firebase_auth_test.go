package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Authorization ヘッダの取り出しが正しいかを検証する。
// VerifyIDToken は Firebase への到達が必要なため、ここではヘッダの
// 形式判定だけを対象にする（不正な形式は client を呼ばずに弾かれる）。
func TestVerifyRejectsMalformedHeader(t *testing.T) {
	// client が nil でも、ヘッダが不正なら参照される前に返る。
	a := NewAuthenticator(nil)

	tests := []struct {
		name   string
		header string
	}{
		{"ヘッダなし", ""},
		{"Bearer がない", "eyJhbGciOi"},
		{"スキームだけ", "Bearer "},
		{"別のスキーム", "Basic eyJhbGciOi"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/me", nil)
			if tt.header != "" {
				r.Header.Set("Authorization", tt.header)
			}

			if _, err := a.verify(r); err == nil {
				t.Fatalf("エラーを期待したが nil が返った")
			}
		})
	}
}

// 不正なヘッダのとき 401 を返し、次のハンドラを呼ばないこと。
func TestRequireBlocksUnauthenticated(t *testing.T) {
	a := NewAuthenticator(nil)

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})

	rec := httptest.NewRecorder()
	a.Require(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/me", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("ステータス = %d, 期待値 = %d", rec.Code, http.StatusUnauthorized)
	}
	if called {
		t.Error("認証に失敗したのに次のハンドラが呼ばれた")
	}
}

// context に載せたトークンを TokenFrom が取り出せること。
// 未設定の context では ok が false になること。
func TestTokenFrom(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/me", nil)

	if _, ok := TokenFrom(r.Context()); ok {
		t.Error("未設定の context で ok = true が返った")
	}

	ctx := withToken(r.Context(), nil)
	if _, ok := TokenFrom(ctx); !ok {
		t.Error("設定済みの context で ok = false が返った")
	}
}
