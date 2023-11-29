package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

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
type SnapshotResponse struct {
	Cmd   string `json:"cmd"`
	Code  int    `json:"code"`
	Value struct {
		Image string `json:"image"`
	} `json:"value"`
}

func login() string {
	url := "http://192.168.52.151/api.cgi?cmd=Login"
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
	fmt.Println("Token = " + tokenName)
	return tokenName
}

func takeSnapshot(tokenName string) {
	url := "http://192.168.52.151/cgi-bin/api.cgi?cmd=Snap&channel=0&rs=wuuPhkmUCeI9WG7C&user=admin&password=&width=640&height=480"
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

func main() {
	// Login to get the token name
	tokenName := login()
	if tokenName == "" {
		fmt.Println("Login failed. Cannot take a snapshot.")
		return
	}

	// Take a snapshot using the obtained token name
	takeSnapshot(tokenName)
}
