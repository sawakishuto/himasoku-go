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
	"github.com/sawakishuto/himasoku-go/internal/handler"
	"github.com/sawakishuto/himasoku-go/internal/handler/middleware"
)

func Run(ctx context.Context) error {
	// SIGINT / SIGTERM を受けたら ctx がキャンセルされる
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client, err := auth.NewFirebaseAuthClient()
	if err != nil {
		return fmt.Errorf("failed to create firebase auth client: %w", err)
	}
	authn := middleware.NewAuthenticator(client)

	// 認証が必要なルートはこちらに登録する。登録するだけで保護される。
	api := http.NewServeMux()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)
	mux.Handle("/", authn.Require(api))

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
