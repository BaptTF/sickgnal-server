package main

import (
	"log"

	"github.com/BaptTF/sickgnal-server/config"
	"github.com/BaptTF/sickgnal-server/server"
	"github.com/BaptTF/sickgnal-server/store"
)

func main() {
	cfg := config.Parse()

	log.Printf("Initializing database at %s", cfg.DBPath)
	db, err := store.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying DB: %v", err)
	}
	defer sqlDB.Close()

	srv := server.New(cfg, db)
	log.Fatal(srv.Run())
}
