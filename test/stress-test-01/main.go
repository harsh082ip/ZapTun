package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

func makeRequest(url string, id int, wg *sync.WaitGroup, completed *int64, failed *int64) {
	defer wg.Done()

	start := time.Now()

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make the request
	resp, err := client.Get(url)
	if err != nil {
		fmt.Printf("Request %d failed: %v\n", id, err)
		atomic.AddInt64(failed, 1)
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Request %d - failed to read body: %v\n", id, err)
		atomic.AddInt64(failed, 1)
		return
	}

	duration := time.Since(start)

	fmt.Printf("Request %d completed:\n", id)
	fmt.Printf("  Status: %s\n", resp.Status)
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Response length: %d bytes\n", len(body))
	fmt.Printf("  Response preview: %.100s...\n", string(body))
	fmt.Println("---")

	// Check if response indicates an error (e.g., subdomain not found)
	responseStr := string(body)
	if resp.StatusCode != 200 ||
		(len(responseStr) > 0 && (responseStr == "subdomain for client_id: kejlkq28 not found in the regisry, or client has disconnected" ||
			responseStr == "subdomain not found in the regisry, or client has disconnected" ||
			responseStr == "client has disconnected" ||
			responseStr == "subdomain not found")) {
		atomic.AddInt64(failed, 1)
	} else {
		atomic.AddInt64(completed, 1)
	}
}

func main() {
	// Define command-line flags
	url := flag.String("url", "", "URL to send requests to")
	numRequests := flag.Int("req", 5, "Number of concurrent requests")
	flag.Parse()

	// Validate URL flag
	if *url == "" {
		fmt.Println("Error: URL is required")
		fmt.Println("Usage: go run main.go -url <URL> [-count <number>]")
		fmt.Println("Example: go run main.go -url http://example.com -count 5")
		return
	}

	var wg sync.WaitGroup
	var completed int64
	var failed int64

	fmt.Printf("Making %d concurrent requests to %s\n\n", *numRequests, *url)

	start := time.Now()

	// Launch goroutines
	for i := 1; i <= *numRequests; i++ {
		wg.Add(1)
		go makeRequest(*url, i, &wg, &completed, &failed)
	}

	// Wait for all requests to complete
	wg.Wait()

	totalDuration := time.Since(start)

	fmt.Printf("Total requests: %d\n", *numRequests)
	fmt.Printf("Completed: %d\n", completed)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Total time: %v\n", totalDuration)
}
