package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var rdb *redis.Client

type CommandRequest struct {
	Command string `json:"command"`
}

type CommandResponse struct {
	Result string `json:"result"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func handleCommand(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Split the command string into separate arguments
	args := strings.Fields(req.Command)

	// Convert []string to []interface{}
	cmdArgs := make([]interface{}, len(args))
	for i, v := range args {
		cmdArgs[i] = v
	}

	// Use the Do method to send the command
	cmd := rdb.Do(ctx, cmdArgs...)
	result, err := cmd.Result()
	if err != nil && err.Error() != "redis: nil" {
		errorResponse := ErrorResponse{Error: err.Error()}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	response := CommandResponse{Result: fmt.Sprintf("%v", result)}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	rdb = redis.NewClient(&redis.Options{
		Addr: "localhost:7369",
	})

	http.Handle("/", http.FileServer(http.Dir("./playground"))) // Serve the static HTML
	http.HandleFunc("/redis-command", handleCommand)            // Endpoint to handle Redis commands

	fmt.Println("Server started at :8080")
	http.ListenAndServe(":8080", nil)
}
