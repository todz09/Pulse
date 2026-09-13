package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// CheckResult holds everything we learn from probing URL.
type CheckResult struct {
	URL         string
	Up         bool
	StatusCode  int
	ResponseTime time.Duration
	Error       error
	CheckAt    time.Time
}

// checkURL sends an HTTP GET request to URL and reports whether it's up.
func checkURL (url string) CheckResult{
	start := time.Now()
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)

	elapsed := time.Since(start)
	
	result := CheckResult{
		URL:         url,
		ResponseTime: elapsed,
		CheckAt:    start,
	}

	if err != nil {
		result.Up = false
		result.Error = err
		return result
	}
	defer resp.Body.Close()

	result.Up = resp.StatusCode >= 200 && resp.StatusCode < 400
	result.StatusCode = resp.StatusCode
	return result
}

func main() {
	urls := []string{
		"https://www.google.com",
		"https://www.github.com",
		"https://www.this-site-does-not-exist-pulse-test.com",
	}

	var wg sync.WaitGroup
	results := make(chan CheckResult, len(urls))

	for _, url := range urls{
		wg.Add(1)
		go func(u string){
			defer wg.Done()
			result := checkURL(u)
			results <- result
		}(url)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results{
		printResult(result)
	}
}

// printResult prints a CheckResult in a human-readable line.
func printResult(result CheckResult) {
	status := "DOWN"
	if result.Up {
		status = "UP"
	}

	if result.Error != nil {
		fmt.Printf("[%s] %s - error: %v\n", status, result.URL, result.Error)
		return
	}

	fmt.Printf("[%s] %s - status: %d - took: %v\n", status, result.URL, result.StatusCode, result.ResponseTime)
}