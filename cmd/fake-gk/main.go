package main

import (
	"log"

	"github.com/nthung2499/fake-gk/internal/config"
	"github.com/nthung2499/fake-gk/internal/db"
	"github.com/nthung2499/fake-gk/internal/server"
)

func main() {
	cfg := config.Load()

	store, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	if err := store.Migrate(); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	router, err := server.New(cfg, store)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	log.Printf("fake-gk listening on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("run server: %v", err)
	}
}
