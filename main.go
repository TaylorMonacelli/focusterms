package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	// Set up the logger
	logFile := &lumberjack.Logger{
		Filename:   "focusterms.log",
		MaxSize:    1, // In megabytes
		MaxBackups: 0,
		MaxAge:     365, // In days
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)

	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting working directory:", err)
		return
	}

	dataPath := filepath.Join(wd, "meta.json")
	dataPath2 := filepath.Join(wd, "meta-b64.txt")

	// Check if file exists
	if _, err := os.Stat(dataPath); err == nil {
		// File exists, delete it
		err := os.Remove(dataPath)
		if err != nil {
			// Error occurred while deleting the file
			panic(err)
		}
	}
	logger.Printf("%s successfully deleted", dataPath)

	// Check if file exists
	if _, err := os.Stat(dataPath2); err == nil {
		// File exists, delete it
		err := os.Remove(dataPath2)
		if err != nil {
			// Error occurred while deleting the file
			panic(err)
		}
	}
	logger.Printf("%s successfully deleted", dataPath2)

	// Make the HTTP request to the metadata service
	url := "http://169.254.169.254/latest/dynamic/instance-identity/document"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Fatalf("Error creating HTTP request: %s", err)
	}

	logger.Printf("Fetching from url: %s", url)
	client := &http.Client{
		Timeout: time.Second * 2,
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Fatalf("Error making HTTP request: %s", err)
	}
	defer resp.Body.Close()

	// Read the response body and parse the JSON data
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Fatalf("Error reading response body: %s", err)
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Fatalf("Error parsing JSON data: %s", err)
	}

	// Pretty print the JSON and write it to a file
	jsonStr, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		logger.Fatalf("Error pretty-printing JSON: %s", err)
	}

	err = ioutil.WriteFile(dataPath, jsonStr, 0o644)
	if err != nil {
		logger.Fatalf("Error writing JSON to file: %s", err)
	}

	base64Str := base64.StdEncoding.EncodeToString(jsonStr)

	// Write the base64-encoded string to a file
	err = ioutil.WriteFile(dataPath2, []byte(base64Str), 0o644)
	if err != nil {
		logger.Fatalf("Error writing base64-encoded JSON to file: %s", err)
	}

	msg := "Successfully fetched instance metadata and wrote it to file"
	msg = fmt.Sprintf("%s %s", msg, dataPath)
	logger.Printf(msg)

	msg = "Successfully fetched instance metadata and wrote it to file"
	msg = fmt.Sprintf("%s %s", msg, dataPath2)
	logger.Printf(msg)

	// Log a success message
	fmt.Printf(base64Str)
}
