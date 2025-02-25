package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// EchoRequest struct is used to receive the message from the client
type EchoRequest struct {
	Message string `json:"message"`
}

// EchoResponse struct is used to return the response in JSON format
type EchoResponse struct {
	Message string `json:"message"`
}

// Handler function for /echo endpoint
func echoHandler(w http.ResponseWriter, r *http.Request) {
	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are supported", http.StatusBadRequest)
		return
	}

	// Decode the request body into EchoRequest struct
	var req EchoRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Failed to parse the request", http.StatusBadRequest)
		return
	}

	// Construct the response
	resp := EchoResponse{
		Message: req.Message,
	}

	// Set the response content type to JSON
	w.Header().Set("Content-Type", "application/json")

	// Encode and write the response
	json.NewEncoder(w).Encode(resp)
}

// Handler function for /home endpoint
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Set the response content type to HTML
	w.Header().Set("Content-Type", "text/html")

	// HTML content with centered text
	htmlContent := `
    <!DOCTYPE html>
    <html>
    <head>
        <title>Edge Matrix Computing</title>
    </head>
    <body style="display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0;">
        <h1>Edge Matrix Computing</h1>
    </body>
    </html>
    `

	// Write the HTML content to the response
	fmt.Fprint(w, htmlContent)
}

func main() {
	// Set up the routes
	http.HandleFunc("/echo", echoHandler)
	http.HandleFunc("/home", homeHandler)

	// Start the server
	fmt.Printf("Server is running, access http://localhost:9527/home\n")
	http.ListenAndServe(":9527", nil)
}
