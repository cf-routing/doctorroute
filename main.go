package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {
	fmt.Println("Starting dr. Route....")
	http.HandleFunc("/start", start)
	http.HandleFunc("/stop", stop)
	http.HandleFunc("/health", health)
	fmt.Println("Doctor route running...")
	log.Fatal(http.ListenAndServe(":"+os.Getenv("PORT"), nil))
}

type Results struct {
	TotalRequests int
	Responses     map[string]int
}

var runResults Results
var polling bool

func health(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Checking health...")
	payload, err := json.Marshal(runResults)
	if err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write(payload)
}

func stop(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Stopping...")
	polling = false
	res.WriteHeader(http.StatusNoContent)
}

func start(res http.ResponseWriter, req *http.Request) {
	fmt.Println("Starting...")
	go func() {
		polling = true
		runResults = Results{}
		runResults.Responses = make(map[string]int)
		for i := 1; polling; i++ {
			fmt.Printf("Poll [%d]...\n", i)

			url := fmt.Sprintf("%s://%s%s", "http", req.Host, "/health")
			resp, err := http.Get(url)
			if err != nil {
				http.Error(res, err.Error(), http.StatusInternalServerError)
			} else {
				count, ok := runResults.Responses[strconv.Itoa(resp.StatusCode)]
				if !ok {
					count = 0
				}
				runResults.Responses[strconv.Itoa(resp.StatusCode)] = count + 1
			}
			runResults.TotalRequests = i
			time.Sleep(1 * time.Second)
		}
	}()
	res.WriteHeader(http.StatusNoContent)
}
