package main

import (
	"log"
	"os"
	"strconv"

	"websitego/internal/config"
	"websitego/internal/database"
	"websitego/internal/migrations"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: go run ./cmd/migrate [up|down|force <version>]")
	}

	action := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := database.NewMySQL(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	switch action {
	case "up":
		if err := migrations.Up(cfg, db); err != nil {
			log.Fatalf("migration up failed: %v", err)
		}
		log.Println("migration up completed")
	case "down":
		if err := migrations.Down(cfg, db); err != nil {
			log.Fatalf("migration down failed: %v", err)
		}
		log.Println("migration down completed")
	case "force":
		if len(os.Args) < 3 {
			log.Fatal("usage: go run ./cmd/migrate force <version>")
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("invalid version: %v", err)
		}
		if err := migrations.Force(cfg, db, version); err != nil {
			log.Fatalf("migration force failed: %v", err)
		}
		log.Printf("migration force completed at version %d\n", version)
	default:
		log.Fatal("unknown action. use: up, down, or force")
	}
}
