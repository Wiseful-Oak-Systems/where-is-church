package seeder

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// MassTimes.org API integration
// API docs: https://apiv4.updateparishdata.org/Default.htm
// Request an API key by contacting MassTimes.org

const massTimesAPIBase = "https://apiv4.updateparishdata.org/Ede65a36a0154d1a930/"

type MassTimesChurch struct {
	ChurchAddressCityName           string  `json:"church_address_city_name"`
	ChurchAddressStreetAddress      string  `json:"church_address_street_address"`
	ChurchAddressPostalCode         string  `json:"church_address_postal_code"`
	ChurchAddressProvinceName       string  `json:"church_address_province_name"`
	ChurchAddressCountryName        string  `json:"church_address_country_territory_name"`
	ChurchName                      string  `json:"church_name"`
	DioceseName                     string  `json:"diocese_name"`
	Latitude                        float64 `json:"latitude"`
	Longitude                       float64 `json:"longitude"`
	PhoneNumber                     string  `json:"phone_number"`
	Email                           string  `json:"email"`
	Website                         string  `json:"website"`
	ChurchTypeName                  string  `json:"church_type_name"`
}

type MassTimesSearchResponse struct {
	Churches []MassTimesChurch `json:"churches"`
}

// SeedFromMassTimes fetches churches from the MassTimes.org API for a given
// country and city, and inserts them into the database.
// Requires a valid API key set in MASSTIMES_API_KEY env var.
func SeedFromMassTimes(db *gorm.DB, apiKey, country, city string) error {
	if apiKey == "" {
		return fmt.Errorf("MASSTIMES_API_KEY is required (request at apiv4.updateparishdata.org)")
	}

	log.Printf("Fetching churches from MassTimes.org: %s, %s...", city, country)

	url := fmt.Sprintf("%sSearchByAddress?address=%s,%s&apikey=%s",
		massTimesAPIBase, city, country, apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("MassTimes API request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("MassTimes API returned %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	var result MassTimesSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode failed: %w", err)
	}

	log.Printf("  Received %d churches from MassTimes.org", len(result.Churches))

	seeds := make([]SeedChurch, 0, len(result.Churches))
	for _, ch := range result.Churches {
		if ch.Latitude == 0 && ch.Longitude == 0 {
			continue
		}
		if ch.ChurchName == "" {
			continue
		}

		address := ch.ChurchAddressStreetAddress
		if ch.ChurchAddressCityName != "" {
			if address != "" {
				address += " - "
			}
			address += ch.ChurchAddressCityName
		}
		if ch.ChurchAddressProvinceName != "" {
			address += " - " + ch.ChurchAddressProvinceName
		}

		seeds = append(seeds, SeedChurch{
			Name:         ch.ChurchName,
			Denomination: "Catholic", // MassTimes.org is Catholic-only
			Address:      address,
			Latitude:     ch.Latitude,
			Longitude:    ch.Longitude,
			Phone:        ch.PhoneNumber,
			Website:      ch.Website,
		})
	}

	return bulkInsertChurches(db, seeds)
}
