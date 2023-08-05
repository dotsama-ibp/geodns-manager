package main

import (
        "bytes"
        "encoding/base64"
        "encoding/json"
        "fmt"
        "io/ioutil"
        "net/http"
        "strings"
        "os"
        "strconv"
        "math"
)

type Record struct {
        ID       string `json:"id"`
        Domain   string `json:"domain"`
        Host     string `json:"host"`
        TTL      string `json:"ttl"`
        Prio     string `json:"prio"`
        Type     string `json:"type"`
        Rdata    string `json:"rdata"`
        EasydnsId string `json:"geozone_id"`
        LastMod  string `json:"last_mod"`
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
        EasydnsId int `json:"easydns_id"`
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
        Domain  string `json:"domain"`
        Host   string `json:"host"`
        Ttl int    `json:"ttl"`
        Prio int    `json:"prio"`
        Type string    `json:"type"`
        Rdata string    `json:"rdata"`
        GeozoneId int `json:"geozone_id"`
}

func main() {
        apiKey := ""
        apiSecret := ""


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
                if member.ServicesAddress != "" && level >= 5 && lat != 0 && long != 0 && active == 1 {
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
        records := loadRecords(apiKey, apiSecret)

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

                payload := Payload{
                  Domain:  "dotters.network",
                  Host:   "sys",
                  Ttl:   60,
                  Prio:   0,
                  Type:   "A",
                  Rdata:   nearestServer,
                  GeozoneId: country.EasydnsId,
                }

                var existing = 0
		var existingId = "";
		var update = 0
                for _, record := range records.Data {
	                recordGeo, _ := strconv.Atoi(record.EasydnsId)		
                	if record.Host == "sys" && recordGeo == country.EasydnsId {
                	  	existing = 1
	        	  	existingId = record.ID
	        	        fmt.Printf("Existing record found %s - %d - %d\n", record.Host, recordGeo, country.EasydnsId)
				if record.Rdata != nearestServer {
					update = 1
				}
                	}
               	}

                // No Record, Create new one
                if existing == 0 {
                        _ = createRecord(apiKey, apiSecret, payload)
	                fmt.Printf("Creating record\n")
                // Record found, update
                } else {
			if update == 1 {
	                        _ = updateRecord(apiKey, apiSecret, payload, existingId)
		                fmt.Printf("Updating record\n")
			}
                }
        }
}

func loadCountries() Countries {
        filePath := "./easydns-countries.json"

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

func loadRecords(apiKey string, apiSecret string) Records {

        client := &http.Client{}
        body := ""

  req, err := http.NewRequest("GET", "https://rest.easydns.net/zones/records/all/dotters.network?format=json", strings.NewReader(body))
  if err != nil {
                fmt.Printf("Failed to create request: %v\n", err)
  }

        auth := fmt.Sprintf("%s:%s", apiKey, apiSecret)
        encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
        req.Header.Set("Authorization", fmt.Sprintf("Basic %s", encodedAuth))
        req.Header.Set("Content-Type", "application/json")

        resp, err := client.Do(req)

        if resp.StatusCode != 200 {
          bodyBytes, _ := ioutil.ReadAll(resp.Body)
          fmt.Errorf("failed to get GeoDNS records: %s", string(bodyBytes))
        }

        bodyBytes, err := ioutil.ReadAll(resp.Body)
        if err != nil {
          fmt.Errorf("failed to get GeoDNS records: %s", string(bodyBytes))
        }

        var records Records
        err = json.Unmarshal(bodyBytes, &records)
        if err != nil {
          fmt.Errorf("failed to get GeoDNS records: %s", string(bodyBytes))
        }

        return records
}

func createRecord(apiKey string, apiSecret string, payload Payload) bool {
        client := &http.Client{}

        payloadBytes, err := json.Marshal(payload)
        if err != nil {
                fmt.Printf("Failed to marshal payload: %v\n", err)
        }

        req, err := http.NewRequest("PUT", "https://rest.easydns.net/zones/records/add/dotters.network/A", bytes.NewBuffer(payloadBytes))
        if err != nil {
                fmt.Printf("Failed to create request: %v\n", err)
        }

        // Set the required headers for authentication
        auth := fmt.Sprintf("%s:%s", apiKey, apiSecret)
        encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
        req.Header.Set("Authorization", fmt.Sprintf("Basic %s", encodedAuth))
        req.Header.Set("Content-Type", "application/json")

        resp, err := client.Do(req)
        if err != nil {
                fmt.Printf("Failed to send request: %v\n", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != 201 {
                fmt.Printf("Failed to get domains: %s\n", resp.Status)
                fmt.Printf("Payload: %s\n", payloadBytes)
                return false
        } else {
                return true
        }
}

func updateRecord(apiKey string, apiSecret string, payload Payload, existingId string) bool {
        client := &http.Client{}

        payloadBytes, err := json.Marshal(payload)
        if err != nil {
                fmt.Printf("Failed to marshal payload: %v\n", err)
        }

        var url = "https://rest.easydns.net/zones/records/" + existingId
        req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
        if err != nil {
                fmt.Printf("Failed to create request: %v\n", err)
        }

        // Set the required headers for authentication
        auth := fmt.Sprintf("%s:%s", apiKey, apiSecret)
        encodedAuth := base64.StdEncoding.EncodeToString([]byte(auth))
        req.Header.Set("Authorization", fmt.Sprintf("Basic %s", encodedAuth))
        req.Header.Set("Content-Type", "application/json")

        resp, err := client.Do(req)
        if err != nil {
                fmt.Printf("Failed to send request: %v\n", err)
        }
        defer resp.Body.Close()

        if resp.StatusCode != 200 {
                fmt.Printf("Failed to update: %s\n", resp.Status)
                fmt.Printf("Payload: %s\n", payloadBytes)
                return false
        }       else {
                return true
        }
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
