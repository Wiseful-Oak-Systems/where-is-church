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
			Denomination: classifyDenomination(el.Tags["denomination"], name),
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

// classifyDenomination determines the denomination from the OSM tag AND the church name.
// When the OSM denomination tag is empty (very common in Brazil), we analyze the name
// for keywords that indicate the denomination. This prevents "Igreja Evangélica"
// from being classified as Catholic.
func classifyDenomination(osmDenom, name string) string {
	// First: use the explicit tag if present
	d := strings.ToLower(strings.TrimSpace(osmDenom))
	if d != "" {
		return mapDenominationTag(d)
	}

	// Second: analyze the church name for denomination clues
	n := strings.ToLower(name)

	// Evangelical / Pentecostal indicators (check first — most common misclassification)
	evangelicalKeywords := []string{
		"evangélica", "evangelica", "evangelical",
		"pentecostal", "assembleia de deus", "assembléia de deus",
		"assembleia", "assembléia",
		"batista", "baptist",
		"adventista", "adventist",
		"universal do reino", "universal",
		"deus é amor", "maranata",
		"quadrangular", "foursquare",
		"metodista livre", "presbiteriana renovada",
		"congregação cristã", "congregação",
		"igreja mundial", "igreja internacional",
		"comunidade evangélica", "comunidade cristã",
		"templo evangélico", "missão evangélica",
		"sara nossa terra", "renascer em cristo",
		"bola de neve", "hillsong",
		"igreja de cristo", "church of christ",
		"igreja do nazareno",
	}
	for _, kw := range evangelicalKeywords {
		if strings.Contains(n, kw) {
			return "Evangelical"
		}
	}

	// Protestant indicators
	protestantKeywords := []string{
		"luterana", "lutheran",
		"presbiteriana", "presbyterian",
		"metodista", "methodist",
		"reformada", "reformed",
		"anglicana", "anglican", "episcopal",
	}
	for _, kw := range protestantKeywords {
		if strings.Contains(n, kw) {
			if strings.Contains(n, "anglicana") || strings.Contains(n, "episcopal") {
				return "Anglican"
			}
			return "Protestant"
		}
	}

	// Orthodox indicators
	orthodoxKeywords := []string{
		"ortodoxa", "orthodox", "bizantina", "antioquena",
	}
	for _, kw := range orthodoxKeywords {
		if strings.Contains(n, kw) {
			return "Orthodox"
		}
	}

	// Catholic indicators (explicit)
	catholicKeywords := []string{
		"paróquia", "paroquia", "parish",
		"catedral", "cathedral",
		"basílica", "basilica",
		"capela", "chapel",
		"mosteiro", "monastery", "convento",
		"santuário", "sanctuary",
		"nossa senhora", "são ", "santa ", "santo ",
		"imaculada", "sagrado coração", "divino",
		"matriz", "igreja católica",
	}
	for _, kw := range catholicKeywords {
		if strings.Contains(n, kw) {
			return "Catholic"
		}
	}

	// If just "Igreja" with no other clues, leave as Other rather than assume Catholic
	if strings.HasPrefix(n, "igreja ") {
		return "Other"
	}

	// Default: Catholic (majority in Brazil, but only for names that don't look Protestant)
	return "Catholic"
}

func mapDenominationTag(d string) string {
	switch {
	case d == "catholic" || d == "roman_catholic" || d == "católica":
		return "Catholic"
	case d == "orthodox" || strings.Contains(d, "orthodox"):
		return "Orthodox"
	case d == "protestant" || d == "lutheran" || d == "reformed" ||
		d == "presbyterian" || d == "methodist" || d == "congregational":
		return "Protestant"
	case d == "anglican" || d == "episcopalian":
		return "Anglican"
	case d == "evangelical" || d == "pentecostal" || d == "baptist" ||
		strings.Contains(d, "assembl") || strings.Contains(d, "universal") ||
		d == "adventist" || d == "neo_pentecostal":
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
