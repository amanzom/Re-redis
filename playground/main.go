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
	Commands []string `json:"commands"`
}

type CommandResponse struct {
	Results []interface{} `json:"results"`
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

	pipe := rdb.Pipeline()
	cmds := make([]*redis.Cmd, len(req.Commands))

	for i, cmdString := range req.Commands {
		// Split the command string into separate arguments
		args := strings.Fields(cmdString)

		// Convert []string to []interface{}
		cmdArgs := make([]interface{}, len(args))
		for j, v := range args {
			cmdArgs[j] = v
		}

		// Queue the command in the pipeline
		cmds[i] = pipe.Do(ctx, cmdArgs...)
	}

	_, err = pipe.Exec(ctx)
	if err != nil && err.Error() != "redis: nil" {
		errorResponse := ErrorResponse{Error: err.Error()}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	results := make([]interface{}, len(cmds))
	for i, cmd := range cmds {
		results[i] = cmd.Val()
	}

	response := CommandResponse{Results: results}
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
