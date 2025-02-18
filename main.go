package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

// APIResponse represents the structure of the JSON response from the API.
type APIResponse struct {
	Connections []Connection `json:"connections"`
}

// Connection represents a single journey between two locations.
type Connection struct {
	From     Stop   `json:"from"`
	To       Stop   `json:"to"`
	Duration string `json:"duration"` // e.g. "00d00:43:00"
}

// Stop holds the departure/arrival information.
type Stop struct {
	Departure string  `json:"departure"` // ISO8601 date-time string
	Arrival   string  `json:"arrival"`
	Station   Station `json:"station"`
}

// Station represents a location such as a train or bus station.
type Station struct {
	Name string `json:"name"`
}

func main() {
	// Check for required command-line arguments.
	if len(os.Args) < 3 {
		fmt.Println("Usage: transport <from> <to>")
		os.Exit(1)
	}

	from := os.Args[1]
	to := os.Args[2]

	// Print a nice ASCII art header with emojis.
	fmt.Println(`

   _____ ____  ____     _____ _      _____
  / ____|  _ \|  _ \   / ____| |    |_   _|
 | (___ | |_) | |_) | | |    | |      | |
  \___ \|  _ <|  _ <  | |    | |      | |
  ____) | |_) | |_) | | |____| |____ _| |_
 |_____/|____/|____/   \_____|______|_____|

`)

	// Build the API URL using the start and destination.
	apiURL := fmt.Sprintf("http://transport.opendata.ch/v1/connections?from=%s&to=%s", from, to)

	// Send the HTTP GET request.
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Fatalf("Error fetching connections: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: received status code %d", resp.StatusCode)
	}

	// Read the response body.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	// Decode the JSON response.
	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	// Check if there are any connections.
	if len(apiResp.Connections) == 0 {
		fmt.Println("No connections found.")
		return
	}

	// Display each connection with ASCII art and emojis.
	for i, conn := range apiResp.Connections {
		fmt.Printf("\nConnection %d:\n", i+1)

		// Parse the departure and arrival times.
		depTime := formatTime(conn.From.Departure)
		arrTime := formatTime(conn.To.Arrival)

		fmt.Printf("🚏  From: %s\n", conn.From.Station.Name)
		if depTime != "" {
			fmt.Printf("    Departure: %s\n", depTime)
		}
		fmt.Println("    │")
		fmt.Println("    ▼")
		fmt.Printf("🚏  To:   %s\n", conn.To.Station.Name)
		if arrTime != "" {
			fmt.Printf("    Arrival:   %s\n", arrTime)
		}
		if conn.Duration != "" {
			fmt.Printf("⏱️  Duration: %s\n", conn.Duration)
		}
		fmt.Println("--------------------------------")
	}
}

// formatTime converts an RFC3339 time string into a "HH:MM" format.
// If parsing fails or the string is empty, it returns the original string.
func formatTime(t string) string {
	if t == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return t
	}
	return parsed.Format("15:04")
}
