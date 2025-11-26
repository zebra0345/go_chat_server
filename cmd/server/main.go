package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-user-server/internal/server"
)

func main() {
	// SIGINT, SIGTERM 잡아서 context 취소
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, ":8080"); err != nil {
		log.Fatalf("server exited with error: %v", err)
	}
}
