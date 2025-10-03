package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

// gossipClient picks 3 random ports in [startPort, endPort]
// and sends HTTP PUT requests with a numeric payload to /gossip every interval seconds.
func gossipClient(startPort, endPort, numNodes, version int) {
	go func() {
		rand.Seed(time.Now().UnixNano())

		// choose 3 random distinct ports
		ports := make(map[int]struct{})
		for len(ports) < numNodes {
			p := rand.Intn(endPort-startPort+1) + startPort
			ports[p] = struct{}{}
		}

		// send to each chosen port
		for port := range ports {
			url := fmt.Sprintf("http://localhost:%d/gossip", port)

			// numeric payload (example: random number)
			payload := []byte(fmt.Sprintf("%d", version))

			req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
			if err != nil {
				log.Printf("Failed to create request for %s: %v", url, err)
				continue
			}
			req.Header.Set("Content-Type", "text/plain")

			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("Failed to contact %s: %v", url, err)
				continue
			}
			resp.Body.Close()
		}

	}()
}

func counterLoop(ch <-chan int, numNodes int) {
	// map[value] = count
	counts := make(map[int]int)

	for val := range ch {
		counts[val]++
		if counts[val] == numNodes {
			fmt.Printf("Committed version %d → count = %d\n", val, counts[val])
		} else {
			fmt.Printf("Received for %d → count = %d\n", val, counts[val])
		}
	}
}

func main() {
	// Read number of goroutines
	numStr := os.Getenv("NUM_GOROUTINES")
	startPortStr := os.Getenv("START_PORT")
	numNodesToSendStr := os.Getenv("NUM_NODES_TO_SEND")
	sendIntervalStr := os.Getenv("SEND_INTERVAL")

	if numStr == "" || startPortStr == "" {
		log.Fatal("Please set NUM_GOROUTINES and START_PORT environment variables")
	}

	num, err := strconv.Atoi(numStr)
	if err != nil {
		log.Fatalf("Invalid NUM_GOROUTINES: %v", err)
	}

	startPort, err := strconv.Atoi(startPortStr)
	if err != nil {
		log.Fatalf("Invalid START_PORT: %v", err)
	}

	numNodesToSend, err := strconv.Atoi(numNodesToSendStr)
	if err != nil {
		log.Fatalf("Invalid NUM_NODES_TO_SEND: %v", err)
	}

	sendInterval, err := strconv.Atoi(sendIntervalStr)
	if err != nil {
		log.Fatalf("Invalid SEND_INTERVAL: %v", err)
	}

	ch := make(chan int)
	go counterLoop(ch, num)

	for i := 0; i <= num; i++ {
		id := i
		port := startPort + id

		// Each goroutine has its own seen map
		seen := make(map[int]bool)

		mux := http.NewServeMux()
		mux.HandleFunc("/gossip", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				http.Error(w, "Only PUT supported", http.StatusMethodNotAllowed)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}
			_ = r.Body.Close()

			val, err := strconv.Atoi(string(body))
			if err != nil {
				http.Error(w, "body must be an integer", http.StatusBadRequest)
				return
			}

			if !seen[val] {
				seen[val] = true
				// Send to shared channel
				ch <- val
				go func() {
					ticker := time.NewTicker(time.Duration(1) * time.Second)
					for i := 0; i < 5; i++ {
						<-ticker.C
						gossipClient(startPort, startPort+num, numNodesToSend, val)
					}
				}()

			} else {
				fmt.Fprintf(w, "Server %d (port %d) already saw %d\n", id, port, val)
			}
		})

		server := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		}

		go func(id, port int, srv *http.Server) {
			log.Printf("Starting server %d on port %d", id, port)
			if err := srv.ListenAndServe(); err != nil {
				log.Printf("Server %d stopped: %v", id, err)
			}
		}(id, port, server)
	}

	go func() {
		ticker := time.NewTicker(time.Duration(sendInterval) * time.Second)
		defer ticker.Stop()
		version := 0
		for {
			<-ticker.C
			version++
			log.Printf("Sending version %d ", version)
			gossipClient(startPort, startPort+num, numNodesToSend, version)
		}
	}()

	// Block forever
	select {}
}
