package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/sawakishuto/himasoku-go/internal/auth"
	"github.com/sawakishuto/himasoku-go/internal/domain/user/usecase"
	"github.com/sawakishuto/himasoku-go/internal/handler"
	"github.com/sawakishuto/himasoku-go/internal/handler/middleware"
	"github.com/sawakishuto/himasoku-go/internal/infra/postgres"
)

func Run(ctx context.Context) error {
	// SIGINT / SIGTERM を受けたら ctx がキャンセルされる
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, err := auth.NewFirebaseAuthClient()
	if err != nil {
		return fmt.Errorf("failed to create firebase auth client: %w", err)
	}

	pool, err := postgres.NewPool(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	authn := middleware.NewAuthenticator(client, postgres.NewIdentityRepository(pool))

	// 登録済みであることまで要求するルートはこちらに登録する。
	// GET /me と POST /users だけは authn.Require で個別に登録する。
	// 未登録の利用者がそこに到達できないと、永遠に登録できないため。
	api := http.NewServeMux()

	mux := http.NewServeMux()
	// ヘルスチェックは資格情報を持たないので保護しない。
	mux.HandleFunc("GET /health", handler.Health)

	// 登録前の利用者が通れる唯一の経路なので、authn.Require 側に置く。
	// Require はトークンを検証するだけで、登録の有無では止めない。
	// RequireRegistered の下に置くと、登録するために登録済みである
	// ことが必要になり、誰も登録できなくなる。
	registration := usecase.NewUserRegistration(postgres.NewUserRepository(pool))
	mux.Handle("POST /users", authn.Require(handler.CreateUser(registration)))

	mux.Handle("/", authn.RequireRegistered(api))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println(err)
		}
	}()
	log.Println("server started on :8080")

	<-ctx.Done()
	stop()
	log.Println("server stopped")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		srv.Close() // タイムアウトしたら強制切断
	}
	log.Println("bye")

	return nil
}
