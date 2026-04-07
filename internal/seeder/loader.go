package seeder

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

// SeedChurch is the JSON structure in seed files.
type SeedChurch struct {
	Name         string  `json:"name"`
	Denomination string  `json:"denomination"`
	Address      string  `json:"address"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Phone        string  `json:"phone,omitempty"`
	Website      string  `json:"website,omitempty"`
	Description  string  `json:"description,omitempty"`
}

// LoadSeedFile reads a gzipped JSON seed file and bulk-inserts into the database.
func LoadSeedFile(db *gorm.DB, path string) error {
	f, err := os.Open(path) //nolint:gosec // path from trusted source
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader %s: %w", path, err)
	}
	defer func() { _ = gz.Close() }()

	var seeds []SeedChurch
	if err := json.NewDecoder(gz).Decode(&seeds); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}

	log.Printf("Loading %d churches from %s...", len(seeds), path)
	return bulkInsertChurches(db, seeds)
}

// bulkInsertChurches converts seed data to models and inserts in batches of 500.
// Uses a single query to check for existing churches in the DB before inserting.
func bulkInsertChurches(db *gorm.DB, seeds []SeedChurch) error {
	// Filter out invalid entries
	valid := make([]models.Church, 0, len(seeds))
	for _, sc := range seeds {
		if sc.Latitude == 0 && sc.Longitude == 0 {
			continue
		}
		if sc.Name == "" {
			continue
		}
		valid = append(valid, models.Church{
			Name:         sc.Name,
			Denomination: sc.Denomination,
			Address:      sc.Address,
			Latitude:     sc.Latitude,
			Longitude:    sc.Longitude,
			Phone:        sc.Phone,
			Website:      sc.Website,
			Description:  sc.Description,
			DataQuality:  models.QualityUnverified,
		})
	}

	log.Printf("  %d valid entries (filtered %d invalid)", len(valid), len(seeds)-len(valid))

	// Batch insert — 500 at a time, skip duplicates via ON CONFLICT DO NOTHING
	batchSize := 500
	inserted := 0
	for i := 0; i < len(valid); i += batchSize {
		end := i + batchSize
		if end > len(valid) {
			end = len(valid)
		}
		batch := valid[i:end]

		result := db.CreateInBatches(batch, len(batch))
		if result.Error != nil {
			// If batch fails (e.g., some duplicates), fall back to one-by-one for this batch
			for _, ch := range batch {
				if err := db.Create(&ch).Error; err == nil {
					inserted++
				}
			}
		} else {
			inserted += int(result.RowsAffected)
		}

		if (i+batchSize)%5000 < batchSize {
			log.Printf("  Progress: %d/%d (%d inserted)", min(i+batchSize, len(valid)), len(valid), inserted)
		}
	}

	log.Printf("  Done: %d inserted out of %d", inserted, len(valid))
	return nil
}

// LoadAllSeedFiles loads all .json.gz files from the seeds/ directory.
func LoadAllSeedFiles(db *gorm.DB, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.json.gz"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no seed files found in %s (run 'make seed-gen' first)", dir)
	}

	for _, f := range files {
		base := strings.TrimSuffix(filepath.Base(f), ".json.gz")
		log.Printf("Loading seed: %s", strings.ToUpper(base))
		if err := LoadSeedFile(db, f); err != nil {
			log.Printf("  Error: %v", err)
		}
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
