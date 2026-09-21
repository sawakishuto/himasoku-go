package main

import (
	"context"
	"log"

	"github.com/sawakishuto/himasoku-go/internal/server"
)

func main() {
	ctx := context.Background()
	if err := server.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
