package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Church struct {
	Name         string  `json:"name"`
	Denomination string  `json:"denomination"`
	Address      string  `json:"address"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Phone        string  `json:"phone,omitempty"`
	Website      string  `json:"website,omitempty"`
	Description  string  `json:"description,omitempty"`
}

type overpassResponse struct {
	Elements []overpassElement `json:"elements"`
}

type overpassElement struct {
	Lat    float64           `json:"lat"`
	Lon    float64           `json:"lon"`
	Center *struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"center,omitempty"`
	Tags map[string]string `json:"tags"`
}

var countries = map[string]string{
	"BR": "Brazil",
	"US": "United States",
	"PT": "Portugal",
	"MX": "Mexico",
	"AR": "Argentina",
	"CO": "Colombia",
	"CL": "Chile",
	"PE": "Peru",
	"IT": "Italy",
	"ES": "Spain",
	"FR": "France",
	"DE": "Germany",
	"PL": "Poland",
	"PH": "Philippines",
	"IE": "Ireland",
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./cmd/seedgen <COUNTRY_CODE> [COUNTRY_CODE...]")
		fmt.Println("\nAvailable countries:")
		for code, name := range countries {
			fmt.Printf("  %s  %s\n", code, name)
		}
		fmt.Println("\nExamples:")
		fmt.Println("  go run ./cmd/seedgen BR          # Brazil only")
		fmt.Println("  go run ./cmd/seedgen BR PT US    # Multiple countries")
		fmt.Println("  go run ./cmd/seedgen ALL         # All countries")
		os.Exit(1)
	}

	codes := os.Args[1:]
	if len(codes) == 1 && strings.ToUpper(codes[0]) == "ALL" {
		codes = make([]string, 0, len(countries))
		for code := range countries {
			codes = append(codes, code)
		}
	}

	outDir := "seeds"
	os.MkdirAll(outDir, 0o750)

	for _, code := range codes {
		code = strings.ToUpper(code)
		name := countries[code]
		if name == "" {
			name = code
		}

		log.Printf("=== Downloading churches for %s (%s) ===", name, code)

		churches, err := downloadCountry(code)
		if err != nil {
			log.Printf("ERROR: %s failed: %v", code, err)
			continue
		}

		outPath := filepath.Join(outDir, strings.ToLower(code)+".json.gz")
		if err := writeGzipJSON(outPath, churches); err != nil {
			log.Printf("ERROR: failed to write %s: %v", outPath, err)
			continue
		}

		log.Printf("  %s: %d churches → %s", code, len(churches), outPath)
	}

	log.Println("Done! Seed files are in the seeds/ directory.")
	log.Println("To load them: make seed-load")
}

func downloadCountry(code string) ([]Church, error) {
	query := fmt.Sprintf(`
[out:json][timeout:300];
area["ISO3166-1"="%s"]->.country;
(
  node["amenity"="place_of_worship"]["religion"="christian"](area.country);
  way["amenity"="place_of_worship"]["religion"="christian"](area.country);
  relation["amenity"="place_of_worship"]["religion"="christian"](area.country);
);
out center tags;
`, code)

	data := url.Values{"data": {strings.TrimSpace(query)}}
	client := &http.Client{Timeout: 10 * time.Minute}

	resp, err := client.PostForm("https://overpass-api.de/api/interpreter", data)
	if err != nil {
		return nil, fmt.Errorf("HTTP failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("overpass returned %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	var result overpassResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	log.Printf("  Received %d elements from OSM", len(result.Elements))

	churches := make([]Church, 0, len(result.Elements))
	for _, el := range result.Elements {
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

		churches = append(churches, Church{
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
	return churches, nil
}

func writeGzipJSON(path string, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	enc := json.NewEncoder(gz)
	return enc.Encode(data)
}

func classifyDenomination(osmDenom, name string) string {
	d := strings.ToLower(strings.TrimSpace(osmDenom))
	if d != "" {
		switch {
		case d == "catholic" || d == "roman_catholic" || d == "católica":
			return "Catholic"
		case d == "orthodox" || strings.Contains(d, "orthodox"):
			return "Orthodox"
		case d == "protestant" || d == "lutheran" || d == "reformed" ||
			d == "presbyterian" || d == "methodist":
			return "Protestant"
		case d == "anglican" || d == "episcopalian":
			return "Anglican"
		case d == "evangelical" || d == "pentecostal" || d == "baptist" ||
			strings.Contains(d, "assembl") || d == "adventist":
			return "Evangelical"
		default:
			return "Other"
		}
	}

	n := strings.ToLower(name)
	evangelicalKW := []string{
		"evangélica", "evangelica", "pentecostal", "assembleia",
		"assembléia", "batista", "adventista", "universal",
		"quadrangular", "congregação cristã", "deus é amor",
		"maranata", "sara nossa terra", "renascer", "bola de neve",
		"igreja mundial", "comunidade evangélica", "templo evangélico",
		"igreja do nazareno", "igreja de cristo",
	}
	for _, kw := range evangelicalKW {
		if strings.Contains(n, kw) {
			return "Evangelical"
		}
	}
	protestantKW := []string{"luterana", "presbiteriana", "metodista", "reformada"}
	for _, kw := range protestantKW {
		if strings.Contains(n, kw) {
			return "Protestant"
		}
	}
	if strings.Contains(n, "anglicana") || strings.Contains(n, "episcopal") {
		return "Anglican"
	}
	if strings.Contains(n, "ortodoxa") || strings.Contains(n, "orthodox") {
		return "Orthodox"
	}
	catholicKW := []string{
		"paróquia", "paroquia", "catedral", "basílica", "capela",
		"mosteiro", "santuário", "nossa senhora", "são ", "santa ",
		"santo ", "matriz", "católica", "imaculada", "sagrado",
	}
	for _, kw := range catholicKW {
		if strings.Contains(n, kw) {
			return "Catholic"
		}
	}
	if strings.HasPrefix(n, "igreja ") {
		return "Other"
	}
	return "Catholic"
}

func buildAddress(tags map[string]string) string {
	parts := []string{}
	if v := tags["addr:street"]; v != "" {
		s := v
		if n := tags["addr:housenumber"]; n != "" {
			s += ", " + n
		}
		parts = append(parts, s)
	}
	if v := tags["addr:suburb"]; v != "" {
		parts = append(parts, v)
	}
	if v := tags["addr:city"]; v != "" {
		parts = append(parts, v)
	}
	if v := tags["addr:state"]; v != "" {
		parts = append(parts, v)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " - ")
	}
	if v := tags["addr:full"]; v != "" {
		return v
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
