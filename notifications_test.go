package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSendNtfyRequest(t *testing.T) {
	title := "Parade completed 👍 & checked"
	message := "Test only: no status update."
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/test-topic" {
			t.Errorf("unexpected destination: %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("title") != title {
			t.Errorf("notification title was not preserved: %q", r.URL.Query().Get("title"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != message {
			t.Errorf("unexpected body %q: %v", body, err)
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "text/plain") {
			t.Errorf("unexpected content type: %s", r.Header.Get("Content-Type"))
		}
		_, _ = io.WriteString(w, `{"event":"message"}`)
	}))
	defer server.Close()
	if err := sendNtfyTo(server.URL+"/test-topic", title, message); err != nil {
		t.Fatal(err)
	}
}

func TestSendNtfyRejectsServerErrors(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusServiceUnavailable} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()
			err := sendNtfyTo(server.URL+"/test-topic", "Test", "Test")
			if err == nil || !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("expected HTTP %d to be reported, got %v", status, err)
			}
		})
	}
}

func TestSendNtfyTimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	defer server.Close()
	err := sendNtfyTo(server.URL+"/test-topic", "Test", "Test")
	var networkError net.Error
	if !errors.As(err, &networkError) || !networkError.Timeout() {
		t.Fatalf("expected a notification timeout, got %v", err)
	}
}

func TestWeekdayBoundaries(t *testing.T) {
	for _, test := range []struct {
		date string
		want bool
	}{
		{"2026-09-11", true},  // Friday
		{"2026-09-12", false}, // Saturday
		{"2026-09-13", false}, // Sunday
		{"2026-09-14", true},  // Monday
	} {
		day, err := time.Parse("2006-01-02", test.date)
		if err != nil {
			t.Fatal(err)
		}
		if got := isWeekDay(day); got != test.want {
			t.Errorf("isWeekDay(%s) = %t, want %t", test.date, got, test.want)
		}
	}
}
