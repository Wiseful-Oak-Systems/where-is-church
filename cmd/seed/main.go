package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wiseful-oak-systems/where-is-church/internal/config"
	"github.com/wiseful-oak-systems/where-is-church/internal/database"
	"github.com/wiseful-oak-systems/where-is-church/internal/models"
)

// Overpass API query to fetch all places of worship in Brazil from OpenStreetMap.
// This returns churches, cathedrals, chapels with name, denomination, address, and coordinates.
const overpassQueryBrazil = `
[out:json][timeout:300];
area["ISO3166-1"="BR"]->.brazil;
(
  node["amenity"="place_of_worship"]["religion"="christian"](area.brazil);
  way["amenity"="place_of_worship"]["religion"="christian"](area.brazil);
  relation["amenity"="place_of_worship"]["religion"="christian"](area.brazil);
);
out center tags;
`

const overpassAPI = "https://overpass-api.de/api/interpreter"

// OverpassResponse is the JSON structure returned by the Overpass API.
type OverpassResponse struct {
	Elements []OverpassElement `json:"elements"`
}

type OverpassElement struct {
	Type   string            `json:"type"`
	ID     int64             `json:"id"`
	Lat    float64           `json:"lat"`
	Lon    float64           `json:"lon"`
	Center *OverpassCenter   `json:"center,omitempty"` // for ways/relations
	Tags   map[string]string `json:"tags"`
}

type OverpassCenter struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func main() {
	log.Println("=== Where is Church? — OSM Data Seeder ===")
	log.Println("Fetching all Christian places of worship in Brazil from OpenStreetMap...")

	cfg := config.Load()
	db := database.Connect(cfg)

	// Fetch from Overpass API
	elements, err := fetchOverpass(overpassQueryBrazil)
	if err != nil {
		log.Fatalf("Failed to fetch from Overpass API: %v", err)
	}
	log.Printf("Received %d elements from OpenStreetMap", len(elements))

	// Convert and insert
	inserted, skipped, errors := 0, 0, 0
	for i, el := range elements {
		lat, lng := el.Lat, el.Lon
		if el.Center != nil {
			lat, lng = el.Center.Lat, el.Center.Lon
		}
		if lat == 0 && lng == 0 {
			skipped++
			continue
		}

		name := el.Tags["name"]
		if name == "" {
			name = el.Tags["official_name"]
		}
		if name == "" {
			skipped++
			continue // Skip unnamed places
		}

		church := models.Church{
			Name:         name,
			Denomination: mapDenomination(el.Tags["denomination"]),
			Address:      buildAddress(el.Tags),
			Latitude:     lat,
			Longitude:    lng,
			Phone:        el.Tags["phone"],
			Website:      el.Tags["website"],
			Description:  el.Tags["description"],
			Verified:     false,
			DataQuality:  models.QualityUnverified,
		}

		// Use FirstOrCreate to avoid duplicates (match on name + approximate location)
		var existing models.Church
		result := db.Where("name = ? AND ABS(latitude - ?) < 0.001 AND ABS(longitude - ?) < 0.001",
			church.Name, church.Latitude, church.Longitude).First(&existing)

		if result.RowsAffected > 0 {
			skipped++
			continue
		}

		if err := db.Create(&church).Error; err != nil {
			errors++
			if errors <= 10 {
				log.Printf("  Error inserting '%s': %v", name, err)
			}
			continue
		}
		inserted++

		if (i+1)%500 == 0 {
			log.Printf("  Progress: %d/%d processed (%d inserted, %d skipped)", i+1, len(elements), inserted, skipped)
		}
	}

	log.Println("=== Seeding Complete ===")
	log.Printf("  Total from OSM: %d", len(elements))
	log.Printf("  Inserted:       %d", inserted)
	log.Printf("  Skipped:        %d (duplicates or unnamed)", skipped)
	log.Printf("  Errors:         %d", errors)

	var total int64
	db.Model(&models.Church{}).Count(&total)
	log.Printf("  Total churches in database: %d", total)
}

func fetchOverpass(query string) ([]OverpassElement, error) {
	data := url.Values{"data": {strings.TrimSpace(query)}}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.PostForm(overpassAPI, data)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("overpass API returned %d: %s", resp.StatusCode, string(body))
	}

	var result OverpassResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Elements, nil
}

// mapDenomination converts OSM denomination tags to our enum values.
func mapDenomination(osmDenom string) string {
	osmDenom = strings.ToLower(strings.TrimSpace(osmDenom))
	switch {
	case osmDenom == "catholic" || osmDenom == "roman_catholic" || osmDenom == "católica" || osmDenom == "":
		return "Catholic" // Default for Brazil (majority Catholic)
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

// buildAddress constructs an address string from OSM tags.
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
		parts = append(parts, "CEP "+v)
	}

	if len(parts) > 0 {
		return strings.Join(parts, " - ")
	}

	// Fallback: use the locality or any available address info
	if v := tags["addr:full"]; v != "" {
		return v
	}
	return ""
}
