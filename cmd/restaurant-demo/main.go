package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	apiURL := env("KITCHEN_API_URL", "http://localhost:8080")
	token := env("RESTAURANT_TOKEN", "demo-pizza-token")
	client := &http.Client{Timeout: 10 * time.Second}

	log.Printf("demo restaurant polling %s", apiURL)
	for {
		process(context.Background(), client, apiURL, token)
		time.Sleep(5 * time.Second)
	}
}

func process(ctx context.Context, client *http.Client, apiURL, token string) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"/api/v1/restaurant/orders?status=new", nil)
	req.Header.Set("X-Restaurant-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("poll error: %v", err)
		return
	}
	defer resp.Body.Close()

	var data struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return
	}

	for _, order := range data.Items {
		for _, status := range []string{"cooking", "delivered"} {
			patch(ctx, client, apiURL, token, order.ID, status)
			time.Sleep(time.Second)
		}
	}
}

func patch(ctx context.Context, client *http.Client, apiURL, token, orderID, status string) {
	body, _ := json.Marshal(map[string]string{"status": status})
	url := fmt.Sprintf("%s/api/v1/restaurant/orders/%s/status", apiURL, orderID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Restaurant-Token", token)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("patch error: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		log.Printf("patch failed: %s", b)
		return
	}
	log.Printf("order %s -> %s", orderID, status)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
