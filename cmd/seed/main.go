package main

import (
	"log"
	"os"

	"github.com/wiseful-oak-systems/where-is-church/internal/config"
	"github.com/wiseful-oak-systems/where-is-church/internal/database"
	"github.com/wiseful-oak-systems/where-is-church/internal/seeder"
)

func main() {
	log.Println("=== Where is Church? — OSM Data Seeder ===")

	cfg := config.Load()
	db := database.Connect(cfg)

	country := cfg.SeedCountry
	if len(os.Args) > 1 {
		country = os.Args[1]
	}

	log.Printf("Seeding churches for country: %s", country)
	if err := seeder.Seed(db, country); err != nil {
		log.Fatalf("Seed failed: %v", err)
	}
}
