//go:build unit

package webhook_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/muratdemir0/gopulse-messages/internal/adapters/ohttp"
	"github.com/muratdemir0/gopulse-messages/internal/adapters/webhook"
)

func TestSend(t *testing.T) {
	cases := []struct {
		name      string
		request   webhook.Request
		response  webhook.Response
		wantErr   bool
		testSetup func(t *testing.T) (webhook.Client, func())
	}{
		{
			name: "Given a valid request, when sending a message, then the message is sent successfully",
			request: webhook.Request{
				To:      "1234567890",
				Content: "Hello, world!",
			},
			response: webhook.Response{
				Message:      "Accepted",
				MessageID:    "1234567890",
				RetryAttempt: 1,
			},
			wantErr: false,
			testSetup: func(t *testing.T) (webhook.Client, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set(ohttp.HeaderRetryAttempt, "1")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"message": "Accepted", "messageId": "1234567890"}`))
				}))
				return *webhook.NewClient(server.URL, ohttp.NewClient()), server.Close
			},
		},
		{
			name: "Given an invalid request, when sending a message, then an error is returned",
			request: webhook.Request{
				To:      "1234567890",
				Content: "Hello, world!",
			},
			wantErr: true,
			testSetup: func(t *testing.T) (webhook.Client, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				return *webhook.NewClient(server.URL, ohttp.NewClient()), server.Close
			},
		},
		{
			name: "Given a valid request but missing retry header, when sending a message, then an error is returned",
			request: webhook.Request{
				To:      "1234567890",
				Content: "Hello, world!",
			},
			wantErr: true,
			testSetup: func(t *testing.T) (webhook.Client, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"message": "Accepted", "messageId": "1234567890"}`))
				}))
				return *webhook.NewClient(server.URL, ohttp.NewClient()), server.Close
			},
		},
		{
			name: "Given a valid request but invalid retry header, when sending a message, then an error is returned",
			request: webhook.Request{
				To:      "1234567890",
				Content: "Hello, world!",
			},
			wantErr: true,
			testSetup: func(t *testing.T) (webhook.Client, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set(ohttp.HeaderRetryAttempt, "abc")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"message": "Accepted", "messageId": "1234567890"}`))
				}))
				return *webhook.NewClient(server.URL, ohttp.NewClient()), server.Close
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, teardown := tc.testSetup(t)
			defer teardown()

			response, err := client.Send(context.TODO(), tc.request, "/messages")
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tc.wantErr && err == nil {
				if response.Message != tc.response.Message {
					t.Errorf("expected message %q, got %q", tc.response.Message, response.Message)
				}
				if response.MessageID != tc.response.MessageID {
					t.Errorf("expected messageID %q, got %q", tc.response.MessageID, response.MessageID)
				}
				if response.RetryAttempt != tc.response.RetryAttempt {
					t.Errorf("expected retry attempt %d, got %d", tc.response.RetryAttempt, response.RetryAttempt)
				}
			}
		})
	}
}
