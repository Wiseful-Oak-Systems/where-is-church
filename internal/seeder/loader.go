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

// SeedChurch is the JSON structure in seed files (matches seedgen output).
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

// LoadSeedFile reads a gzipped JSON seed file and inserts churches into the database.
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

	var churches []SeedChurch
	if err := json.NewDecoder(gz).Decode(&churches); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}

	log.Printf("Loading %d churches from %s...", len(churches), path)

	inserted, skipped := 0, 0
	for i, sc := range churches {
		if sc.Latitude == 0 && sc.Longitude == 0 {
			skipped++
			continue
		}

		church := models.Church{
			Name:         sc.Name,
			Denomination: sc.Denomination,
			Address:      sc.Address,
			Latitude:     sc.Latitude,
			Longitude:    sc.Longitude,
			Phone:        sc.Phone,
			Website:      sc.Website,
			Description:  sc.Description,
			DataQuality:  models.QualityUnverified,
		}

		var existing models.Church
		if db.Where("name = ? AND ABS(latitude - ?) < 0.001 AND ABS(longitude - ?) < 0.001",
			church.Name, church.Latitude, church.Longitude).First(&existing).RowsAffected > 0 {
			skipped++
			continue
		}

		if err := db.Create(&church).Error; err != nil {
			skipped++
			continue
		}
		inserted++

		if (i+1)%2000 == 0 {
			log.Printf("  Progress: %d/%d (%d inserted)", i+1, len(churches), inserted)
		}
	}

	log.Printf("Loaded %s: %d inserted, %d skipped", path, inserted, skipped)
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
