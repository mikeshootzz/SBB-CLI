package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// APIResponse represents the JSON response from the transport API.
type APIResponse struct {
	Connections []Connection `json:"connections"`
}

// Connection represents an overall journey.
type Connection struct {
	From     Stop      `json:"from"`
	To       Stop      `json:"to"`
	Duration string    `json:"duration"` // e.g., "00d00:55:00"
	Sections []Section `json:"sections"`
}

// Section represents one leg (step) of a journey.
type Section struct {
	Departure Stop     `json:"departure"`
	Arrival   Stop     `json:"arrival"`
	Journey   *Journey `json:"journey"` // may be nil for a walking transfer
}

// Journey holds information about the transportation used in a section.
type Journey struct {
	Category string `json:"category"` // e.g., "S" or "IR"
	Number   string `json:"number"`   // e.g., "14" or "36"
	Operator string `json:"operator"` // not used in display
	To       string `json:"to"`       // final destination of this leg
}

// Stop holds the details for a departure or arrival.
type Stop struct {
	Departure string     `json:"departure"` // ISO8601 time string
	Arrival   string     `json:"arrival"`   // ISO8601 time string
	Platform  string     `json:"platform"`  // planned platform
	Station   Station    `json:"station"`
	Prognosis *Prognosis `json:"prognosis,omitempty"`
}

// Station represents a station or stop.
type Station struct {
	Name string `json:"name"`
}

// Prognosis holds the realtime information (if available) for a stop.
type Prognosis struct {
	Platform    string `json:"platform"`
	Arrival     string `json:"arrival"`
	Departure   string `json:"departure"`
	Capacity1st string `json:"capacity1st"`
	Capacity2nd string `json:"capacity2nd"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: transport <from> <to>")
		os.Exit(1)
	}
	from := os.Args[1]
	to := os.Args[2]

	// Header with ASCII art and emojis.
	fmt.Println(`
   ____  _     _       ____           _
  / ___|| |_  (_) ___ |  _ \ ___  ___| |_ ___  _ __
  \___ \| __| | |/ __|| |_) / _ \/ __| __/ _ \| '__|
   ___) | |_  | |\__ \|  _ <  __/\__ \ || (_) | |
  |____/ \__| |_||___/|_| \_\___||___/\__\___/|_|

🚆  Welcome to Transport CLI 🚏
`)

	// Build the API URL.
	apiURL := fmt.Sprintf("http://transport.opendata.ch/v1/connections?from=%s&to=%s", from, to)
	resp, err := http.Get(apiURL)
	if err != nil {
		log.Fatalf("Error fetching connections: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error: received status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	if len(apiResp.Connections) == 0 {
		fmt.Println("No connections found.")
		return
	}

	// Display each connection.
	for i, conn := range apiResp.Connections {
		fmt.Printf("\nConnection %d: Overall Duration: %s\n", i+1, formatDurationString(conn.Duration))
		if len(conn.Sections) > 0 {
			fmt.Print(displayFancyTimeline(conn.Sections))
		} else {
			// Fallback if no sections available.
			fmt.Printf("%s\n", formatStopFancy(conn.From, true))
			fmt.Printf("  ──( Walk )──▶\n")
			fmt.Printf("%s\n", formatStopFancy(conn.To, false))
		}
		fmt.Println("--------------------------------")
	}
}

// displayFancyTimeline builds a multi-line, left-to-right timeline for the connection's sections.
func displayFancyTimeline(sections []Section) string {
	var builder strings.Builder
	// Print the first stop using its departure details.
	builder.WriteString(formatStopFancy(sections[0].Departure, true) + "\n")
	// For each section, print the journey and the arrival stop.
	for _, sec := range sections {
		builder.WriteString("    " + formatJourneyFancy(sec.Journey) + "\n")
		builder.WriteString(formatStopFancy(sec.Arrival, false) + "\n")
	}
	return builder.String()
}

// formatStopFancy returns a formatted string for a stop including a warning if needed.
// isDeparture flag indicates whether this is a departure (true) or arrival (false) stop.
func formatStopFancy(stop Stop, isDeparture bool) string {
	var t string
	if isDeparture {
		t = formatTimeString(stop.Departure)
	} else {
		t = formatTimeString(stop.Arrival)
	}
	return fmt.Sprintf("[ %s (%s | Plat %s%s) ]",
		stop.Station.Name,
		t,
		stop.Platform,
		warningSymbol(stop, isDeparture),
	)
}

// warningSymbol returns a warning emoji if the stop’s prognosis differs from the schedule.
// (For example, if the departure/arrival time or platform has changed.)
func warningSymbol(stop Stop, isDeparture bool) string {
	if stop.Prognosis != nil {
		if isDeparture && stop.Prognosis.Departure != "" && stop.Prognosis.Departure != stop.Departure {
			return " ⚠️"
		}
		if !isDeparture && stop.Prognosis.Arrival != "" && stop.Prognosis.Arrival != stop.Arrival {
			return " ⚠️"
		}
		if stop.Prognosis.Platform != "" && stop.Prognosis.Platform != stop.Platform {
			return " ⚠️"
		}
	}
	return ""
}

// formatJourneyFancy returns a formatted string for a journey segment, omitting any internal id.
func formatJourneyFancy(journey *Journey) string {
	if journey == nil {
		return "──( Walk )──▶"
	}
	// Display only the category and line number (e.g., "S 14")
	return fmt.Sprintf("──( %s %s )──▶", journey.Category, journey.Number)
}

// formatTimeString converts an ISO8601 time string to a "15:04" format.
func formatTimeString(t string) string {
	if t == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, t)
	if err != nil {
		// Try an alternative layout if the timezone is formatted as +0100 (without colon).
		parsed, err = time.Parse("2006-01-02T15:04:05-0700", t)
		if err != nil {
			return t // return the original if parsing fails
		}
	}
	return parsed.Format("15:04")
}

// formatDurationString converts a duration like "00d00:55:00" into a human-friendly string.
func formatDurationString(dur string) string {
	// Expected format: "00d00:55:00" => days 'd' then HH:MM:SS.
	parts := strings.SplitN(dur, "d", 2)
	if len(parts) != 2 {
		return dur
	}
	daysStr := parts[0]
	timePart := parts[1]
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		return dur
	}
	tparts := strings.Split(timePart, ":")
	if len(tparts) != 3 {
		return dur
	}
	hours, err := strconv.Atoi(tparts[0])
	if err != nil {
		return dur
	}
	minutes, err := strconv.Atoi(tparts[1])
	if err != nil {
		return dur
	}
	seconds, err := strconv.Atoi(tparts[2])
	if err != nil {
		return dur
	}
	var partsOut []string
	if days > 0 {
		if days == 1 {
			partsOut = append(partsOut, fmt.Sprintf("%d day", days))
		} else {
			partsOut = append(partsOut, fmt.Sprintf("%d days", days))
		}
	}
	if hours > 0 {
		if hours == 1 {
			partsOut = append(partsOut, fmt.Sprintf("%d hour", hours))
		} else {
			partsOut = append(partsOut, fmt.Sprintf("%d hours", hours))
		}
	}
	if minutes > 0 {
		if minutes == 1 {
			partsOut = append(partsOut, fmt.Sprintf("%d minute", minutes))
		} else {
			partsOut = append(partsOut, fmt.Sprintf("%d minutes", minutes))
		}
	}
	// Only show seconds if no other unit is significant.
	if seconds > 0 && days == 0 && hours == 0 && minutes == 0 {
		if seconds == 1 {
			partsOut = append(partsOut, fmt.Sprintf("%d second", seconds))
		} else {
			partsOut = append(partsOut, fmt.Sprintf("%d seconds", seconds))
		}
	}
	if len(partsOut) == 0 {
		return "0 minutes"
	}
	return strings.Join(partsOut, " ")
}
