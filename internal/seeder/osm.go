package seeder

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/wiseful-oak-systems/where-is-church/internal/models"
	"gorm.io/gorm"
)

const overpassAPI = "https://overpass-api.de/api/interpreter"

// OverpassQuery returns the query to fetch all Christian places of worship
// for the given ISO 3166-1 country code.
func OverpassQuery(countryCode string) string {
	return fmt.Sprintf(`
[out:json][timeout:300];
area["ISO3166-1"="%s"]->.country;
(
  node["amenity"="place_of_worship"]["religion"="christian"](area.country);
  way["amenity"="place_of_worship"]["religion"="christian"](area.country);
  relation["amenity"="place_of_worship"]["religion"="christian"](area.country);
);
out center tags;
`, countryCode)
}

type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
}

type overpassElement struct {
	Type   string            `json:"type"`
	ID     int64             `json:"id"`
	Lat    float64           `json:"lat"`
	Lon    float64           `json:"lon"`
	Center *overpassCenter   `json:"center,omitempty"`
	Tags   map[string]string `json:"tags"`
}

type overpassCenter struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// SeedIfEmpty loads seed files from seeds/ directory if available,
// otherwise falls back to live Overpass API download.
func SeedIfEmpty(db *gorm.DB, countryCode string) {
	var count int64
	db.Model(&models.Church{}).Count(&count)
	if count > 0 {
		log.Printf("Database has %d churches — skipping seed", count)
		return
	}

	go func() {
		// Try loading from local seed file first (fast, reliable)
		seedFile := fmt.Sprintf("seeds/%s.json.gz", strings.ToLower(countryCode))
		if _, err := os.Stat(seedFile); err == nil {
			log.Printf("Found seed file %s — loading...", seedFile)
			if err := LoadSeedFile(db, seedFile); err != nil {
				log.Printf("Seed file load failed: %v — falling back to API", err)
			} else {
				return
			}
		}

		// Fallback: download from Overpass API
		log.Println("No seed file found — downloading from OpenStreetMap...")

		// Phase 1: São Paulo metro area first (fast)
		log.Println("Phase 1: Seeding São Paulo metro area...")
		spQuery := `
[out:json][timeout:120];
(
  node["amenity"="place_of_worship"]["religion"="christian"](-24.1,-47.2,-23.2,-46.2);
  way["amenity"="place_of_worship"]["religion"="christian"](-24.1,-47.2,-23.2,-46.2);
);
out center tags;
`
		if err := seedFromQuery(db, spQuery); err != nil {
			log.Printf("São Paulo seed failed: %v", err)
		}

		// Phase 2: Rest of the country
		log.Printf("Phase 2: Seeding rest of %s...", countryCode)
		if err := Seed(db, countryCode); err != nil {
			log.Printf("Full seed failed: %v (run 'make seed-gen' then 'make seed-load')", err)
		}
	}()
}

func seedFromQuery(db *gorm.DB, query string) error {
	elements, err := fetchOverpass(query)
	if err != nil {
		return err
	}
	return insertElements(db, elements)
}

// Seed fetches all churches from OSM for the given country and inserts them.
func Seed(db *gorm.DB, countryCode string) error {
	query := OverpassQuery(countryCode)
	elements, err := fetchOverpass(query)
	if err != nil {
		return fmt.Errorf("overpass fetch failed: %w", err)
	}
	return insertElements(db, elements)
}

func insertElements(db *gorm.DB, elements []overpassElement) error {
	log.Printf("Processing %d elements from OpenStreetMap...", len(elements))

	seeds := make([]SeedChurch, 0, len(elements))
	for _, el := range elements {
		lat, lng := el.Lat, el.Lon
		if el.Center != nil {
			lat, lng = el.Center.Lat, el.Center.Lon
		}
		if lat == 0 && lng == 0 {
			continue
		}
		name := el.Tags["name"]
		if name == "" {
			name = el.Tags["official_name"]
		}
		if name == "" {
			continue
		}
		seeds = append(seeds, SeedChurch{
			Name:         name,
			Denomination: mapDenomination(el.Tags["denomination"]),
			Address:      buildAddress(el.Tags),
			Latitude:     lat,
			Longitude:    lng,
			Phone:        el.Tags["phone"],
			Website:      el.Tags["website"],
			Description:  el.Tags["description"],
		})
	}

	return bulkInsertChurches(db, seeds)
}

func fetchOverpass(query string) ([]overpassElement, error) {
	data := url.Values{"data": {strings.TrimSpace(query)}}
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.PostForm(overpassAPI, data)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("overpass returned %d: %s", resp.StatusCode, string(body))
	}

	var result overpassResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}
	return result.Elements, nil
}

func mapDenomination(osmDenom string) string {
	osmDenom = strings.ToLower(strings.TrimSpace(osmDenom))
	switch {
	case osmDenom == "catholic" || osmDenom == "roman_catholic" || osmDenom == "católica" || osmDenom == "":
		return "Catholic"
	case osmDenom == "orthodox" || strings.Contains(osmDenom, "orthodox"):
		return "Orthodox"
	case osmDenom == "protestant" || osmDenom == "lutheran" || osmDenom == "reformed" ||
		osmDenom == "presbyterian" || osmDenom == "methodist" || osmDenom == "congregational":
		return "Protestant"
	case osmDenom == "anglican" || osmDenom == "episcopalian":
		return "Anglican"
	case osmDenom == "evangelical" || osmDenom == "pentecostal" || osmDenom == "baptist" ||
		strings.Contains(osmDenom, "assembl") || strings.Contains(osmDenom, "universal") ||
		osmDenom == "adventist" || osmDenom == "neo_pentecostal":
		return "Evangelical"
	default:
		return "Other"
	}
}

func buildAddress(tags map[string]string) string {
	parts := []string{}
	if v := tags["addr:street"]; v != "" {
		street := v
		if n := tags["addr:housenumber"]; n != "" {
			street += ", " + n
		}
		parts = append(parts, street)
	}
	if v := tags["addr:suburb"]; v != "" {
		parts = append(parts, v)
	} else if v := tags["addr:neighbourhood"]; v != "" {
		parts = append(parts, v)
	}
	if v := tags["addr:city"]; v != "" {
		parts = append(parts, v)
	}
	if v := tags["addr:state"]; v != "" {
		parts = append(parts, v)
	}
	if v := tags["addr:postcode"]; v != "" {
		parts = append(parts, "CEP " + v)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " - ")
	}
	if v := tags["addr:full"]; v != "" {
		return v
	}
	return ""
}
