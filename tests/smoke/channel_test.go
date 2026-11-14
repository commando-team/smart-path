//go:build smoke

package smoke

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type smokeChannel struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	IsActive       bool      `json:"isActive"`
	FeePercent     int64     `json:"feePercent"`
	SuccessPercent int64     `json:"successPercent"`
	LatencyMs      int64     `json:"latencyMs"`
}

func TestChannelCRUD(t *testing.T) {
	baseURL := os.Getenv("SMOKE_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	client := &http.Client{}

	createBody := `{"name":"Smoke Channel ` + uuid.New().String() + `","isActive":true,"feePercent":2,"successPercent":100,"latencyMs":50}`
	created := smokeChannel{}
	postJSON(t, client, baseURL+"/channels", http.MethodPost, createBody, http.StatusCreated, &created)
	if created.ID == uuid.Nil || created.Name == "" {
		t.Fatalf("unexpected created channel: %+v", created)
	}
	if !created.IsActive || created.FeePercent != 2 || created.SuccessPercent != 100 || created.LatencyMs != 50 {
		t.Fatalf("created channel has unexpected fields: %+v", created)
	}

	var loaded smokeChannel
	getJSON(t, client, fmt.Sprintf("%s/channels/%s", baseURL, created.ID), http.StatusOK, &loaded)
	if loaded.ID != created.ID || loaded.Name != created.Name {
		t.Fatalf("GET returned unexpected channel: %+v", loaded)
	}

	updateBody := `{"name":"Updated Smoke Channel","isActive":false,"feePercent":5,"successPercent":80,"latencyMs":300}`
	var updated smokeChannel
	postJSON(t, client, fmt.Sprintf("%s/channels/%s", baseURL, created.ID), http.MethodPut, updateBody, http.StatusOK, &updated)
	if updated.ID != created.ID || updated.Name != "Updated Smoke Channel" || updated.IsActive {
		t.Fatalf("unexpected updated channel: %+v", updated)
	}
	if updated.FeePercent != 5 || updated.SuccessPercent != 80 || updated.LatencyMs != 300 {
		t.Fatalf("updated channel has unexpected fields: %+v", updated)
	}
}

func postJSON(t *testing.T, client *http.Client, url, method, body string, wantStatus int, result any) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("request %s %s: got HTTP %d, want %d", method, url, resp.StatusCode, wantStatus)
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		t.Fatalf("decode %s response: %v", method, err)
	}
}

func getJSON(t *testing.T, client *http.Client, url string, wantStatus int, result any) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("GET %s: got HTTP %d, want %d", url, resp.StatusCode, wantStatus)
	}
	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
}
