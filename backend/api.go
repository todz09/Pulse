package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MonitorStatus is what we send back to the frontend: a monitor plus its latest check.
type MonitorStatus struct {
	ID             int        `json:"id"`
	Name           string     `json:"name"`
	URL            string     `json:"url"`
	Up             *bool      `json:"up"`
	StatusCode     *int       `json:"status_code"`
	ResponseTimeMs *int       `json:"response_time_ms"`
	CheckedAt      *time.Time `json:"checked_at"`
}

// CheckHistoryItem is one row of history for a single monitor.
type CheckHistoryItem struct {
	Up             bool       `json:"up"`
	StatusCode     *int       `json:"status_code"`
	ResponseTimeMs *int       `json:"response_time_ms"`
	ErrorMessage   *string    `json:"error_message"`
	CheckedAt      time.Time `json:"checked_at"`
}

// enableCORS wraps a handler so browsers on other domains (like our React app) are allowed to call it.
func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

// handleGetMonitors returns every monitor along with its most recent check result.
func handleGetMonitors(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		rows, err := pool.Query(ctx, `
			SELECT m.id, m.name, m.url, c.up, c.status_code, c.response_time_ms, c.checked_at
			FROM monitors m
			LEFT JOIN LATERAL (
				SELECT up, status_code, response_time_ms, checked_at
				FROM checks
				WHERE monitor_id = m.id
				ORDER BY checked_at DESC
				LIMIT 1
			) c ON true
			ORDER BY m.id
		`)
		if err != nil {
			log.Println("query error:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var statuses []MonitorStatus
		for rows.Next() {
			var s MonitorStatus
			err := rows.Scan(&s.ID, &s.Name, &s.URL, &s.Up, &s.StatusCode, &s.ResponseTimeMs, &s.CheckedAt)
			if err != nil {
				log.Println("scan error:", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			statuses = append(statuses, s)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statuses)
	}
}

// handleGetMonitorChecks returns the last 50 checks for one specific monitor.
func handleGetMonitorChecks(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid monitor id", http.StatusBadRequest)
			return
		}

		rows, err := pool.Query(ctx, `
			SELECT up, status_code, response_time_ms, error_message, checked_at
			FROM checks
			WHERE monitor_id = $1
			ORDER BY checked_at DESC
			LIMIT 50
		`, id)
		if err != nil {
			log.Println("query error:", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var history []CheckHistoryItem
		for rows.Next() {
			var c CheckHistoryItem
			err := rows.Scan(&c.Up, &c.StatusCode, &c.ResponseTimeMs, &c.ErrorMessage, &c.CheckedAt)
			if err != nil {
				log.Println("scan error:", err)
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			history = append(history, c)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)
	}
}

// startServer sets up routes and starts listening on the given port.
func startServer(pool *pgxpool.Pool, port string) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/monitors", enableCORS(handleGetMonitors(pool)))
	mux.HandleFunc("GET /api/monitors/{id}/checks", enableCORS(handleGetMonitorChecks(pool)))

	log.Printf("API server listening on port %s\n", port)
	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatal("Server failed: ", err)
	}
}