//go:build unit
package ohttp_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/muratdemir0/gopulse-messages/internal/adapters/ohttp"
)

func TestClientDo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := ohttp.NewClient()
	req, err := http.NewRequest("PUT", server.URL, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("PUT request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status code 202, got %d", resp.StatusCode)
	}
}