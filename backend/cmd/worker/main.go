package main

import (
	"context"
	"log"
	"time"

	"github.com/example/synergyflow/backend/internal/app"
)

func main() {
	ctx := context.Background()
	srv, err := app.New(ctx, app.LoadConfig())
	if err != nil {
		log.Fatal(err)
	}
	log.Println("SynergyFlow worker started")
	for {
		if err := srv.ProcessEmailJobs(ctx); err != nil {
			log.Println("worker:", err)
		}
		time.Sleep(10 * time.Second)
	}
}
