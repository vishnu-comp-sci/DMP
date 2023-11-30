package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Config struct to hold configuration parameters
type Config struct {
	IPAddress               string `json:"ipAddress"`
	Username                string `json:"username"`
	Password                string `json:"password"`
	MotionDurationMinutes   int    `json:"motionDurationMinutes"`
	MotionCheckIntervalSecs int    `json:"motionCheckIntervalSeconds"`
}

// LoginResponse struct for the login API response
type LoginResponse struct {
	Cmd   string `json:"cmd"`
	Code  int    `json:"code"`
	Value struct {
		Token struct {
			LeaseTime int    `json:"leaseTime"`
			Name      string `json:"name"`
		} `json:"Token"`
	} `json:"value"`
}

// MotionStateResponse struct for the motion detection state API response
type MotionStateResponse struct {
	Cmd   string `json:"cmd"`
	Code  int    `json:"code"`
	Value struct {
		State int `json:"state"`
	} `json:"value"`
}

// SnapshotResponse struct for the snapshot API response
type SnapshotResponse struct {
	Cmd   string `json:"cmd"`
	Code  int    `json:"code"`
	Value struct {
		Image string `json:"image"`
	} `json:"value"`
}

func loadConfig() (Config, error) {
	file, err := ioutil.ReadFile("config.json")
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

// Mutex to protect the flag
var flagMutex sync.Mutex

// Flag to control motion detection checking
var motionCheckFlag = true

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	// Define a flag to control motion detection checking
	stopFlag := flag.Bool("stop", false, "Stop motion detection checking")
	// Parse the command line arguments
	flag.Parse()
	if *stopFlag {
		fmt.Println("Motion detection checking stopped by user.")
		return
	}

	// Construct Login URLs
	apiUrlLogin := fmt.Sprintf("http://%s/api.cgi?cmd=Login", config.IPAddress)
	//apiUrlSnapshot := fmt.Sprintf("http://%s/cgi-bin/api.cgi?cmd=Snap&channel=0&rs=wuuPhkmUCeI9WG7C&user=admin&password=&width=640&height=480", config.IPAddress)
	apiUrlSnapshot := fmt.Sprintf("http://%s/cgi-bin/api.cgi?cmd=Snap&channel=0&rs=wuuPhkmUCeI9WG7C&user=%s&password=%s&width=640&height=480", config.IPAddress, config.Username, config.Password)
	apiUrlMotion := fmt.Sprintf("http://%s/api.cgi?cmd=GetMdState&user=%s&password=%s", config.IPAddress, config.Username, config.Password)
	// Login to get the token name
	fmt.Print(apiUrlSnapshot)
	//fmt.Print
	tokenName := login(apiUrlLogin)
	if tokenName == "" {
		fmt.Println("Login failed. Cannot take a snapshot.")
		return
	}

	// Take a snapshot using the obtained token name
	fmt.Println("\n")
	fmt.Println("Token = " + tokenName)
	//Get screenshot
	takeSnapshot(apiUrlSnapshot, tokenName)
	// Check motion detection state for the specified duration
	go checkMotion(apiUrlMotion, config.MotionDurationMinutes, config.MotionCheckIntervalSecs, stopFlag)

	// Wait for user to stop motion detection checking (press Enter)
	fmt.Println("Press Enter to stop motion detection checking...")
	fmt.Scanln()

	// Stop the motion detection checking by updating the flag
	flagMutex.Lock()
	motionCheckFlag = false
	flagMutex.Unlock()

	fmt.Println("Stopping motion detection checking. Please wait for the current check to complete.")
}

func login(url string) string {
	method := "POST"

	payload := strings.NewReader(`[{
	"cmd": "Login",
	"param": { "User": {
	"Version": "0", "userName": "admin", "password": ""
	} }
	}]`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return ""
	}
	req.Header.Add("Content-Type", "text/plain")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return ""
	}

	var loginResponse []LoginResponse
	err = json.Unmarshal(body, &loginResponse)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return ""
	}

	// Extract the token name
	tokenName := loginResponse[0].Value.Token.Name
	return tokenName
}

func takeSnapshot(url, tokenName string) {
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return
	}

	// Add token to the request header
	req.Header.Add("Cookie", "name="+tokenName)

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fmt.Println("Snapshot request failed with status code:", res.StatusCode)
		return
	}

	// Read the image response
	imgData, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading image response:", err)
		return
	}

	// Save the image to a file
	err = ioutil.WriteFile("snapshot.jpg", imgData, 0644)
	if err != nil {
		fmt.Println("Error saving snapshot:", err)
		return
	}

	fmt.Println("Snapshot saved to snapshot.jpg")
}

func checkMotion(url string, durationMinutes, intervalSeconds int, stopFlag *bool) {
	fmt.Printf("Checking motion detection for %d minutes at %d-second intervals...\n", durationMinutes, intervalSeconds)

	endTime := time.Now().Add(time.Duration(durationMinutes) * time.Minute)

	for time.Now().Before(endTime) {
		// Check the stop flag
		flagMutex.Lock()
		if !motionCheckFlag {
			flagMutex.Unlock()
			fmt.Println("Motion detection checking stopped.")
			return
		}
		flagMutex.Unlock()

		// Check motion detection state
		state := getMotionState(url)

		// Print the motion state
		if state == 0 {
			fmt.Println("No motion detected.")
		} else {
			fmt.Println("Motion detected!")
		}

		// Sleep for the specified interval
		time.Sleep(time.Duration(intervalSeconds) * time.Second)
	}

	fmt.Println("Motion detection check complete.")
}

func getMotionState(url string) int {
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return -1
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return -1
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fmt.Println("Motion state request failed with status code:", res.StatusCode)
		return -1
	}

	// Read the motion state response
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading motion state response:", err)
		return -1
	}

	var motionStateResponse []MotionStateResponse
	err = json.Unmarshal(body, &motionStateResponse)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return -1
	}

	// Extract the motion state
	state := motionStateResponse[0].Value.State
	return state
}
