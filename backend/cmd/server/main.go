package main

import (
	"context"
	"log"
	"os"

	"github.com/example/synergyflow/backend/internal/app"
)

func main() {
	ctx := context.Background()
	srv, err := app.New(ctx, app.LoadConfig())
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.Run(os.Getenv("PORT")); err != nil {
		log.Fatal(err)
	}
}
