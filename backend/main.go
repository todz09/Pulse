package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// Monitor represents one row from the "monitors" table.
type Monitor struct {
	ID   int
	URL  string
	Name string
}

// CheckResult holds everything we learn from probing one URL.
type CheckResult struct {
	MonitorID    int
	URL          string
	Up           bool
	StatusCode   int
	ResponseTime time.Duration
	Error        error
	CheckedAt    time.Time
}

func checkURL(m Monitor) CheckResult {
	start := time.Now()

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(m.URL)

	elapsed := time.Since(start)

	result := CheckResult{
		MonitorID:    m.ID,
		URL:          m.URL,
		ResponseTime: elapsed,
		CheckedAt:    start,
	}

	if err != nil {
		result.Up = false
		result.Error = err
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Up = resp.StatusCode >= 200 && resp.StatusCode < 400

	return result
}

func printResult(r CheckResult) {
	status := "DOWN"
	if r.Up {
		status = "UP"
	}
	if r.Error != nil {
		fmt.Printf("[%s] %s - error: %v\n", status, r.URL, r.Error)
		return
	}
	fmt.Printf("[%s] %s - status: %d - took: %v\n", status, r.URL, r.StatusCode, r.ResponseTime)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("Unable to connect to database: ", err)
	}
	defer pool.Close()

	// Start the HTTP API in the background
	go startServer(pool, port)

	// Run one check cycle immediately on startup
	runCheckCycle(ctx, pool)

	// Then run every 5 minutes, forever
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		runCheckCycle(ctx, pool)
	}
}

// runCheckCycle fetches all monitors, checks them concurrently, and saves results.
func runCheckCycle(ctx context.Context, pool *pgxpool.Pool) {
	monitors, err := fetchMonitors(ctx, pool)
	if err != nil {
		fmt.Println("Unable to fetch monitors:", err)
		return
	}

	fmt.Printf("\n--- Checking %d monitors at %s ---\n", len(monitors), time.Now().Format(time.RFC1123))

	var wg sync.WaitGroup
	results := make(chan CheckResult, len(monitors))

	for _, m := range monitors {
		wg.Add(1)
		go func(mon Monitor) {
			defer wg.Done()
			result := checkURL(mon)
			results <- result
		}(m)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		printResult(result)
		err := saveCheck(ctx, pool, result)
		if err != nil {
			fmt.Printf("  failed to save check for %s: %v\n", result.URL, err)
		}
	}
}

// fetchMonitors reads every row from the "monitors" table.
func fetchMonitors(ctx context.Context, pool *pgxpool.Pool) ([]Monitor, error) {
	rows, err := pool.Query(ctx, "SELECT id, url, name FROM monitors")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []Monitor
	for rows.Next() {
		var m Monitor
		err := rows.Scan(&m.ID, &m.URL, &m.Name)
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	return monitors, nil
}

// saveCheck inserts one check result into the "checks" table.
func saveCheck(ctx context.Context, pool *pgxpool.Pool, r CheckResult) error {
	var errMsg *string
	if r.Error != nil {
		msg := r.Error.Error()
		errMsg = &msg
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO checks (monitor_id, up, status_code, response_time_ms, error_message)
		 VALUES ($1, $2, $3, $4, $5)`,
		r.MonitorID, r.Up, r.StatusCode, r.ResponseTime.Milliseconds(), errMsg,
	)
	return err
}