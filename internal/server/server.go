package server

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/sawakishuto/himasoku-go/internal/handler"
)

func Run(ctx context.Context) error {
	// SIGINT / SIGTERM を受けたら ctx がキャンセルされる
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handler.Health)

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
