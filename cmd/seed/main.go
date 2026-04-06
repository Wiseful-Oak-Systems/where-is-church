package main

import (
	"log"
	"os"

	"github.com/wiseful-oak-systems/where-is-church/internal/config"
	"github.com/wiseful-oak-systems/where-is-church/internal/database"
	"github.com/wiseful-oak-systems/where-is-church/internal/seeder"
)

func main() {
	log.Println("=== Where is Church? — Data Seeder ===")

	cfg := config.Load()
	db := database.Connect(cfg)

	// If seed files exist, load them. Otherwise download from OSM.
	if err := seeder.LoadAllSeedFiles(db, "seeds"); err != nil {
		log.Printf("No seed files found: %v", err)

		country := cfg.SeedCountry
		if len(os.Args) > 1 {
			country = os.Args[1]
		}

		log.Printf("Downloading from OpenStreetMap for %s...", country)
		if err := seeder.Seed(db, country); err != nil {
			log.Fatalf("Seed failed: %v", err)
		}
	}
}
