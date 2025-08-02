//go:build unit
package webhook_test

import (
	"context"
	"github.com/muratdemir0/gopulse-messages/internal/adapters/ohttp"
	"github.com/muratdemir0/gopulse-messages/internal/adapters/webhook"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSend(t *testing.T) {
	cases := []struct{
		name string
		request webhook.Request
		response webhook.Response
		wantErr bool
		testSetup func(t *testing.T) (webhook.Client, func())
	}{
		{
			name: "Given a valid request, when sending a message, then the message is sent successfully",
			request: webhook.Request{
				To: "1234567890",
				Content: "Hello, world!",
			},
			response: webhook.Response{
				Message: "Hello, world!",
			},
			wantErr: false,
			testSetup: func(t *testing.T) (webhook.Client, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"message": "Hello, world!", "messageId": "1234567890"}`))
				}))
				return *webhook.NewClient(server.URL, ohttp.NewClient()), server.Close
			},
		},
		{
			name: "Given an invalid request, when sending a message, then an error is returned",
			request: webhook.Request{
				To: "1234567890",
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
			}
		})
	}
}
