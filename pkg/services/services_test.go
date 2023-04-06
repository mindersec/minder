package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	pb "github.com/stacklok/mediator/pkg/generated/protobuf/go/proto/v1"
	"golang.org/x/oauth2"
)

func TestCheckHealth(t *testing.T) {
	server := NewServer(nil)

	req := &pb.HealthRequest{}
	resp, err := server.CheckHealth(context.Background(), req)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedStatus := "OK"
	if resp.Status != expectedStatus {
		t.Errorf("Expected status %q, got %q", expectedStatus, resp.Status)
	}
}

func TestAuthUrl(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer ts.Close()

	oauth2Config := &oauth2.Config{
		RedirectURL: ts.URL,
	}

	server := NewServer(oauth2Config)
	req := &pb.AuthUrlRequest{}
	resp, err := server.AuthUrl(context.Background(), req)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !strings.HasPrefix(resp.Url, "https://github.com/login/oauth/authorize") {
		t.Errorf("Expected URL to start with 'https://github.com/login/oauth/authorize', got %q", resp.Url)
	}
}
