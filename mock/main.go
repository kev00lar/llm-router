package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"
	"log"
)

type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func main() {
	http.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		if rand.Float32() < 0.2 {
			log.Println("Mock Provider: Simulating 500 Internal Server Error")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := ChatResponse{
			ID:      "chatcmpl-mock123",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "mock-gpt-4",
		}
		resp.Choices = append(resp.Choices, struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		}{})
		resp.Choices[0].Message.Role = "assistant"
		resp.Choices[0].Message.Content = "This is a successful response from the Mock LLM."
		resp.Choices[0].FinishReason = "stop"

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	log.Println("Mock LLM Service starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
