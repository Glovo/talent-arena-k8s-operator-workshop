package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type RequestPayload struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ResponsePayload struct {
	Response string `json:"response"`
}

// check llm model with success prompt
func checkMyModel(ctx context.Context, addr, model, prompt string) (bool, error) {
	log := log.FromContext(ctx)
	// Create JSON payload
	payload := RequestPayload{
		Model:  model,
		Prompt: "Answer 'yes' or 'no'. " + prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	// Make HTTP request
	url := fmt.Sprintf("http://%s/api/generate", addr)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("ollama API returned non-200 status code: %d", resp.StatusCode)
	}

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// Parse JSON response
	var result ResponsePayload
	if err := json.Unmarshal(body, &result); err != nil {
		return false, err
	}
	log.Info("Ollama response", "response", result.Response)

	return strings.Contains(strings.ToLower(result.Response), "yes"), nil
}
