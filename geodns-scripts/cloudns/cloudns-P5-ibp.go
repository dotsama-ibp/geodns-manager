package main

import (
        "fmt"
        "net/http"
        "net/url"
        "strings"
	"io/ioutil"
	"encoding/json"
	"strconv"
	"os"
	"math"
)

type Record struct {
        ID       string `json:"id"`
        Host     string `json:"host"`
        TTL      string `json:"ttl"`
        Type     string `json:"type"`
        Record    string `json:"record"`
        GeodnsId string `json:"geodns-location"`
}

type Records struct {
        TM   int64    `json:"tm"`
        Data []Record `json:"data"`
}

type Country struct {
        Name     string `json:"name"`
        CC     string `json:"country_code"`
        Lat    string `json:"latitude"`
        Long string `json:"longitude"`
        GeodnsId int `json:"geodns-id"`
}

type Countries struct {
        Country []Country `json:"countries"`
}

type Member struct {
        Name           string            `json:"name"`
        Website        string            `json:"website"`
        Logo           string            `json:"logo"`
        Membership     string            `json:"membership"`
        CurrentLevel   string            `json:"current_level"`
        Active   string            `json:"active"`
        LevelTimestamp map[string]string `json:"level_timestamp"`
        ServicesAddress string            `json:"services_address"`
        Region         string            `json:"region"`
        Lat                          string            `json:"latitude"`
        Long                          string            `json:"longitude"`
        Payments       map[string]Payment `json:"payments"`
}

type Payment struct {
        ValidatorAddress string `json:"validator_address"`
        PaymentAddress   string `json:"payment_address"`
        Signature        string `json:"signature"`
}

type Members struct {
        Members map[string]Member `json:"members"`
}

type Payload struct {
        Apikey  string `json:"auth-id"`
        Pass  string `json:"auth-password"`
        Domain  string `json:"domain-name"`
        Host   string `json:"host"`
        Ttl int    `json:"ttl"`
        Type string    `json:"record-type"`
        Record string    `json:"record"`
        GeozoneId int `json:"geodns-location"`
}

func main() {
	apiKey := ""
	apiSecret := ""
	domain := "ibp.network"
	host := "testing-p5"
	minLevel := 5

        // Load Member JSON File
        members := loadMembers()

        var count1 = 0
        var count2 = 0
        var validMembers []Member
        for _, member := range members.Members {
                count1++
                level, _ := strconv.Atoi(member.CurrentLevel)
                active, _ := strconv.Atoi(member.Active)
                lat, _ := strconv.ParseFloat(member.Lat, 64)
                long, _ := strconv.ParseFloat(member.Long, 64)
                if member.ServicesAddress != "" && level >= minLevel && lat != 0 && long != 0 && active == 1 {
                        count2++
                        validMembers = append(validMembers, member)
                }
        }

        fmt.Printf("Loaded %d valid members from a total of %d\n", count2, count1)

        // Load Countries JSON File
        countries := loadCountries()

        var count = 0
        for _, country := range countries.Country {
                count++
		fmt.Printf("Loaded country: %s\n", country.Name)
        }

        fmt.Printf("Loaded countries: %i\n", count)

        // Get DNS Records
        records := loadRecords(apiKey, apiSecret, domain)

        // Assign countries to members
        for _, country := range countries.Country {
               minDistance := math.MaxFloat64
               nearestServer := ""

                for _, member := range validMembers {
                        memberLat, _ := strconv.ParseFloat(member.Lat, 64)
                        memberLong, _ := strconv.ParseFloat(member.Long, 64)
                        countryLat, _ := strconv.ParseFloat(country.Lat, 64)
                        countryLong, _ := strconv.ParseFloat(country.Long, 64)
                        distance := getDistance(countryLat, countryLong, memberLat, memberLong)

                        fmt.Printf("Country: %s testing %s - Distance: %f\n", country.Name, member.Name, distance)

                        if distance < minDistance {
                                minDistance = distance
                                nearestServer = member.ServicesAddress
                        }
                }

                fmt.Printf("Country: %s assigned to %s - Distance: %f\n", country.Name, nearestServer, minDistance)

                var existing = 0
                var existingId = "";
                var update = 0
                for _, record := range records {
                        recordGeo, _ := strconv.Atoi(record.GeodnsId)
                        if record.Host == host && recordGeo == country.GeodnsId {
                                existing = 1
                                existingId = record.ID
                                fmt.Printf("Existing record found %s - %d - %d\n", record.Host, recordGeo, country.GeodnsId)
                                if record.Record != nearestServer {
                                        update = 1
                                }
                        }
                }

		GeoId := strconv.Itoa(country.GeodnsId)

                // No Record, Create new one
                if existing == 0 {
                        _ = createRecord(apiKey, apiSecret, domain, host, "60", "A", nearestServer, GeoId)
                        fmt.Printf("Creating record\n")
                // Record found, update
                } else {
                        if update == 1 {
                                _ = updateRecord(apiKey, apiSecret, existingId, domain, host, "60", nearestServer, GeoId)
                                fmt.Printf("Updating record %s\n", existingId)
                        }
                }
        }
}

func loadCountries() Countries {
        filePath := "./cloudns-countries.json"

        fileContents, err := ioutil.ReadFile(filePath)
        if err != nil {
                fmt.Printf("Error reading file: %v\n", err)
                os.Exit(1)
        }

        var countries Countries
        err = json.Unmarshal(fileContents, &countries)
        if err != nil {
                fmt.Printf("Error unmarshalling JSON: %v\n", err)
                os.Exit(1)
        }
        return countries
}

func loadMembers() Members {
        filePath := "./members.json"

        fileContents, err := ioutil.ReadFile(filePath)
        if err != nil {
                fmt.Printf("Error reading file: %v\n", err)
                os.Exit(1)
        }

        var members Members
        err = json.Unmarshal(fileContents, &members)
        if err != nil {
                fmt.Printf("Error unmarshalling JSON: %v\n", err)
                os.Exit(1)
        }
        return members
}

func createRecord(apiKey string, apiSecret string, domain string, host string, ttl string, dnstype string, record string, geozoneid string) bool {
        client := &http.Client{}

	data := url.Values{}
	data.Set("sub-auth-user", apiKey)
	data.Set("auth-password", apiSecret)
	data.Set("domain-name", domain)
	data.Set("record-type", dnstype)
	data.Set("host", host)
	data.Set("record", record)
	data.Set("ttl", ttl)
	data.Set("geodns-location", geozoneid)

        req, err := http.NewRequest("POST", "https://api.cloudns.net/dns/add-record.json", strings.NewReader(data.Encode()))
        if err != nil {
                fmt.Printf("Failed to create request: %v\n", err)
        }
	
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

        resp, err := client.Do(req)
        if err != nil {
                fmt.Printf("Failed to send request: %v\n", err)
        }
        defer resp.Body.Close()

return true
}

func loadRecords(apiKey string, apiSecret string, domain string) []Record {

        client := &http.Client{}
        data := url.Values{}
        data.Set("sub-auth-user", apiKey)
        data.Set("auth-password", apiSecret)
        data.Set("domain-name", domain)

	req, err := http.NewRequest("POST", "https://api.cloudns.net/dns/records.json", strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
	}

        req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

        resp, err := client.Do(req)
        if resp.StatusCode != 200 {
          bodyBytes, _ := ioutil.ReadAll(resp.Body)
          fmt.Errorf("failed to get GeoDNS records: %s", string(bodyBytes))
        }

        bodyBytes, err := ioutil.ReadAll(resp.Body)
        if err != nil {
          fmt.Errorf("failed to get GeoDNS records: %s", string(bodyBytes))
        }

	var recordsMap map[string]Record
	if err := json.Unmarshal([]byte(bodyBytes), &recordsMap); err != nil {
		fmt.Printf("Error unmarshaling JSON: %v\n", err)
	}

	var records []Record
	for _, record := range recordsMap {
		records = append(records, record)
	}

return records
}

func updateRecord(apiKey string, apiSecret string, id string, domain string, host string, ttl string, record string, geozoneid string) bool {
        client := &http.Client{}
        data := url.Values{}
        data.Set("sub-auth-user", apiKey)
        data.Set("auth-password", apiSecret)
        data.Set("domain-name", domain)
        data.Set("record-id", id)
        data.Set("host", host)
        data.Set("record", record)
        data.Set("ttl", ttl)
        data.Set("geodns-location", geozoneid)

        req, err := http.NewRequest("POST", "https://api.cloudns.net/dns/mod-record.json", strings.NewReader(data.Encode()))
        if err != nil {
                fmt.Printf("Failed to create request: %v\n", err)
        }

        req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

        resp, err := client.Do(req)
        if resp.StatusCode != 200 {
          bodyBytes, _ := ioutil.ReadAll(resp.Body)
          fmt.Errorf("failed to get GeoDNS records: %s", string(bodyBytes))
        }

        bodyBytes, err := ioutil.ReadAll(resp.Body)
        if err != nil {
          fmt.Errorf("failed to get GeoDNS records: %s", string(bodyBytes))
        }

return true
}

func getDistance(lat1, lon1, lat2, lon2 float64) float64 {
        const R = 6371 // Earth's radius in km
        dLat := (lat2 - lat1) * (math.Pi / 180)
        dLon := (lon2 - lon1) * (math.Pi / 180)

        a := math.Sin(dLat/2)*math.Sin(dLat/2) +
                math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180))*
                        math.Sin(dLon/2)*math.Sin(dLon/2)
        c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

        return R * c
}
