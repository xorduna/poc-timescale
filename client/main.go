package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	Interval   time.Duration
	Batch      time.Duration
	NumDevices int
	APIURL     string
}

type Metric struct {
	Ts       time.Time `json:"ts"`
	Temp     float64   `json:"temp"`
	AmbHumid float64   `json:"amb_humid"`
	Setpoint float64   `json:"setpoint"`
	AmbTemp  float64   `json:"amb_temp"`
	Coverage float64   `json:"coverage"`
}

type Payload struct {
	AssetID string   `json:"asset_id"`
	Metrics []Metric `json:"metrics"`
}

func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "m") || strings.HasSuffix(s, "h") || strings.HasSuffix(s, "s") {
		return time.ParseDuration(s)
	}
	return 0, fmt.Errorf("invalid duration format: %s", s)
}

func main() {
	intervalStr := flag.String("interval", "5m", "Interval for generating metrics")
	batchStr := flag.String("batch", "5m", "Interval for sending metrics")
	numDevices := flag.Int("num", 1, "Number of fake devices")
	apiURL := flag.String("api", "http://localhost:8080", "Base URL of the API")
	flag.Parse()

	interval, err := parseDuration(*intervalStr)
	if err != nil {
		log.Fatalf("Invalid interval: %v", err)
	}

	batch, err := parseDuration(*batchStr)
	if err != nil {
		log.Fatalf("Invalid batch: %v", err)
	}

	config := Config{
		Interval:   interval,
		Batch:      batch,
		NumDevices: *numDevices,
		APIURL:     *apiURL,
	}

	var wg sync.WaitGroup
	for i := 0; i < config.NumDevices; i++ {
		wg.Add(1)
		go runDevice(config, &wg)
	}

	wg.Wait()
}

func runDevice(config Config, wg *sync.WaitGroup) {
	defer wg.Done()
	drift := time.Duration(rand.Int63n(int64(config.Batch)))

	log.Printf("Starting device with interval %v batch %v drift %v", config.Interval, config.Batch, drift)

	assetID := uuid.New().String()
	time.Sleep(drift)

	ticker := time.NewTicker(config.Interval)
	batchTicker := time.NewTicker(config.Batch)

	var metrics []Metric
	var mu sync.Mutex

	// Initialize the metrics with some starting values
	temp := 20.0 + rand.Float64()*10.0
	ambHumid := 50.0 + rand.Float64()*20.0
	setpoint := 22.0 + rand.Float64()*4.0
	ambTemp := 18.0 + rand.Float64()*10.0

	for {
		select {
		case t := <-ticker.C:
			// Generate new metric
			temp = generateSmoothedValue(temp, 0.1, 18, 32)
			ambHumid = generateSmoothedValue(ambHumid, 0.1, 30, 80)
			setpoint = generateSmoothedValue(setpoint, 0.05, 20, 26)
			ambTemp = generateSmoothedValue(ambTemp, 0.1, 15, 35)

			metric := Metric{
				Ts:       t,
				Temp:     temp,
				AmbHumid: ambHumid,
				Setpoint: setpoint,
				AmbTemp:  ambTemp,
			}

			mu.Lock()

			//log.Printf("Generated metric: %v", metric)

			metrics = append(metrics, metric)
			mu.Unlock()

		case <-batchTicker.C:
			// Send metrics
			if len(metrics) > 0 {
				mu.Lock()
				payload := Payload{
					AssetID: assetID,
					Metrics: metrics,
				}
				metrics = nil // Clear the metrics
				mu.Unlock()

				go sendMetrics(config, payload)
				log.Printf("Sent %d metrics", len(payload.Metrics))
			}
		}
	}
}

func generateSmoothedValue(current, volatility, min, max float64) float64 {
	change := (rand.Float64() - 0.5) * volatility
	newValue := current + change
	if newValue < min {
		return min
	}
	if newValue > max {
		return max
	}
	return newValue
}

func sendMetrics(config Config, payload Payload) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling payload: %v", err)
		return
	}

	resp, err := http.Post(fmt.Sprintf("%s/assets/%s/metrics", config.APIURL, payload.AssetID), "application/json", strings.NewReader(string(jsonPayload)))
	if err != nil {
		log.Printf("Error sending metrics: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Printf("Unexpected status code: %d", resp.StatusCode)
	}
}
