package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

func checkHealth(key string) error {
	return probeModels(&http.Client{Timeout: 5 * time.Second}, "http://127.0.0.1:8317/v1/models", key)
}

func probeModels(client *http.Client, endpoint, key string) error {
	if key == "" {
		return errors.New("subscription service key is missing")
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return fmt.Errorf("create subscription health request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+key)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("contact subscription service: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("subscription service returned status %d", resp.StatusCode)
	}

	var models struct {
		Data []json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&models); err != nil {
		return fmt.Errorf("read subscription models: %w", err)
	}
	if len(models.Data) == 0 {
		return errors.New("subscription service has no available models")
	}
	return nil
}
