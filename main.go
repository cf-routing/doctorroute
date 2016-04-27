package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
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

type Poller interface {
	Poll(uri string) int
}

type httpPoller struct {
}

type tcpPoller struct {
}

func (h *httpPoller) Poll(url string) int {
	var statusCode int
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error connecting to app: %s\n", err.Error())
		statusCode = 500
	} else {
		statusCode = resp.StatusCode
	}
	return statusCode
}

func (h *tcpPoller) Poll(endpoint string) int {
	conn, err := net.DialTimeout("tcp", endpoint, 5*time.Second)
	if err != nil {
		fmt.Printf("Error connecting to app: %s\n", err.Error())
		return 500
	}

	defer conn.Close()
	message := []byte(fmt.Sprintf("Time is %d", time.Now().Nanosecond()))
	_, err = conn.Write(message)
	if err != nil {
		return 500
	}

	buff := make([]byte, 1024)
	n, err := conn.Read(buff)
	if err != nil || n <= 0 {
		return 500
	}
	return 200
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

		// i.e. http://foo.com/health or foo.com:9000
		url := startRequest.Endpoint

		var poller Poller
		if strings.HasPrefix(url, "http://") {
			poller = &httpPoller{}
		} else {
			poller = &tcpPoller{}
		}

		fmt.Println("Endpoint to poll", url)
		go func() {
			polling = true
			runResults = Results{}
			runResults.Responses = make(map[string]int)
			for i := 1; polling; i++ {
				fmt.Printf("Poll [%d]...\n", i)
				statusCode := poller.Poll(url)
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
