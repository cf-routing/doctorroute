package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

type StartRequest struct {
	Endpoint string
}

var runResults Results
var polling bool

func health(res http.ResponseWriter, req *http.Request) {
	if req.Method == "GET" {
		fmt.Println("Checking health...")
		payload, err := json.Marshal(runResults)
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusOK)
		res.Write(payload)
	} else {
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func stop(res http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		fmt.Println("Stopping...")
		polling = false
		res.WriteHeader(http.StatusNoContent)
	} else {
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func start(res http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		fmt.Println("Starting...")
		var startRequest StartRequest
		payload, err := ioutil.ReadAll(req.Body)
		if err != nil {
			fmt.Println("Error while readin request", err.Error())
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		err = json.Unmarshal(payload, &startRequest)
		if err != nil {
			fmt.Println("Error while decoding request", err.Error())
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		if startRequest.Endpoint == "" {
			startRequest.Endpoint = req.Host
		}
		url := fmt.Sprintf("%s", startRequest.Endpoint)
		fmt.Println("Endpoint to poll", url)
		go func() {
			polling = true
			runResults = Results{}
			runResults.Responses = make(map[string]int)
			for i := 1; polling; i++ {
				var statusCode int

				fmt.Printf("Poll [%d]...\n", i)
				resp, err := http.Get(url)
				if err != nil {
					fmt.Printf("Error connecting to app: %s\n", err.Error())
					statusCode = 500
				} else {
					statusCode = resp.StatusCode
				}

				count, ok := runResults.Responses[strconv.Itoa(statusCode)]
				if !ok {
					count = 0
				}
				runResults.Responses[strconv.Itoa(statusCode)] = count + 1
				runResults.TotalRequests = i
				time.Sleep(1 * time.Second)
			}
		}()
		res.WriteHeader(http.StatusNoContent)
	} else {
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}
