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

const InternalServerError = "500"

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
	Poll(uri string) string
}

type httpPoller struct {
}

type tcpPoller struct {
}

func (h *httpPoller) Poll(url string) string {
	var statusCode string
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error connecting to app: %s\n", err.Error())
		statusCode = InternalServerError
	} else {
		statusCode = strconv.Itoa(resp.StatusCode)
	}
	return statusCode
}

func (h *tcpPoller) Poll(endpoint string) string {
	conn, err := net.DialTimeout("tcp", endpoint, 5*time.Second)
	if err != nil {
		fmt.Printf("Error connecting to app: %s\n", err.Error())
		return InternalServerError
	}

	defer conn.Close()
	message := []byte(fmt.Sprintf("GET /health HTTP/1.1\nHost: %s\n\n", endpoint))
	_, err = conn.Write(message)
	if err != nil {
		fmt.Printf("Error writing HTTP req: %s\n", err.Error())
		return InternalServerError
	}

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil || n <= 0 {
		fmt.Printf("Error reading HTTP response: %s\n", err.Error())
		return InternalServerError
	}

	body := string(buf[:n])

	fmt.Printf("body: %s\n", body)

	parts := strings.Split(body, " ")
	if len(parts) > 1 {
		statusCode := parts[1]
		fmt.Printf("statusCode: %s\n", statusCode)
		return statusCode
	}

	return InternalServerError
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
		if polling {
			fmt.Println("Stopping previous run...")
			polling = false
		}
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
				count, ok := runResults.Responses[statusCode]
				if !ok {
					count = 0
				}
				runResults.Responses[statusCode] = count + 1
				runResults.TotalRequests = i
				time.Sleep(1 * time.Second)
			}
		}()
		res.WriteHeader(http.StatusNoContent)
	} else {
		res.WriteHeader(http.StatusMethodNotAllowed)
	}
}
