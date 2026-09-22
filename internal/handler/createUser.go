package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/sawakishuto/himasoku-go/internal/domain/user/entity"
	"github.com/sawakishuto/himasoku-go/internal/domain/user/usecase"
	"github.com/sawakishuto/himasoku-go/internal/handler/middleware"
)

// CreateUser は POST /users を処理する。
//
// authn.Require の下に置く前提。未登録の利用者が通れる唯一の経路なので、
// RequireRegistered の下に置くと誰も登録できなくなる。
func CreateUser(registration usecase.UserRegistration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := middleware.TokenFrom(r.Context())
		if !ok {
			// Require を通っていれば必ず入っている。無いのは配線の誤り。
			log.Print("token is missing: POST /users must sit behind authn.Require")
			http.Error(w, `{"error":"Internal Server Error"}`, http.StatusInternalServerError)
			return
		}

		var body struct {
			DisplayName string `json:"display_name"`
			Email       string `json:"email"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
			return
		}

		err := registration.Execute(r.Context(), &usecase.RegisterUserInput{
			// 誰であるかは本人確認の結果だけを使う。本文から取ると、
			// 他人の sub を名乗って登録できてしまう。
			Identity:    entity.Identity{Provider: token.Provider, Subject: token.Subject},
			DisplayName: body.DisplayName,
			Email:       body.Email,
		})
		switch {
		case errors.Is(err, usecase.ErrInvalidInput):
			http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		case err != nil:
			log.Printf("failed to register user: %v", err)
			http.Error(w, `{"error":"Internal Server Error"}`, http.StatusInternalServerError)
		default:
			// 再送でも 201 を返す。register_user は冪等で、既に登録済みなら
			// 既存の行に落ち着く。区別する情報をクライアントは使わない。
			w.WriteHeader(http.StatusCreated)
		}
	}
}
