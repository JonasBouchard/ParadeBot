package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

func TestParadeSaveVerification(t *testing.T) {
	pw, err := playwright.Run()
	if err != nil {
		t.Skipf("browser verification tests need the Playwright driver installed: %v", err)
	}
	defer pw.Stop()
	browser, err := pw.Chromium.Launch()
	if err != nil {
		t.Skipf("browser verification tests need Chromium installed: %v", err)
	}
	defer browser.Close()
	const oldTimestamp = "09-SEP @ 05:30:00 - Test - 1"
	const newTimestamp = "09-SEP @ 05:31:00 - Test - 1"
	const comment = "Test & comment\nsecond line"
	for _, test := range []struct {
		name            string
		ack             string
		saveStatus      int
		readbackStatus  string
		readbackComment string
		readbackTime    string
		readbackFailure bool
		loginRedirect   bool
		fakeSave        bool
		wantVerified    bool
	}{
		{name: "persisted save", wantVerified: true},
		{name: "server rejects save", saveStatus: 500},
		{name: "HTTP 200 without acknowledgement", ack: `{}`},
		{name: "HTTP 200 with login HTML", ack: `<html>Sign in</html>`},
		{name: "invalid acknowledgement type", ack: `{"last_updated":true}`},
		{name: "unchanged acknowledgement", ack: `{"last_updated":"` + oldTimestamp + `"}`},
		{name: "UI updates but save not persisted", readbackTime: oldTimestamp},
		{name: "wrong persisted status", readbackStatus: "2"},
		{name: "wrong persisted comment", readbackComment: "different"},
		{name: "another save replaced ours", readbackTime: "09-SEP @ 05:32:00 - Test - 1"},
		{name: "readback HTTP error", readbackFailure: true},
		{name: "readback session expired", loginRedirect: true},
		{name: "unrelated response and fake UI success", fakeSave: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var posts, reads atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					if r.URL.Path == "/status/new_status" {
						posts.Add(1)
						if err := r.ParseForm(); err != nil || r.Form.Get("status") != "1" || r.Form.Get("comment") != comment {
							t.Errorf("submitted fields changed: %v", err)
						}
					}
					w.Header().Set("Content-Type", "application/json")
					if test.saveStatus != 0 {
						w.WriteHeader(test.saveStatus)
					}
					if test.ack != "" {
						fmt.Fprint(w, test.ack)
					} else {
						_ = json.NewEncoder(w).Encode(map[string]string{"last_updated": newTimestamp})
					}
					return
				}
				if r.URL.Path != "/status" {
					http.NotFound(w, r)
					return
				}
				isReadback := reads.Add(1) > 1
				if isReadback && test.readbackFailure {
					http.Error(w, "Unavailable", http.StatusServiceUnavailable)
					return
				}
				if isReadback && test.loginRedirect {
					fmt.Fprint(w, "<html>Sign in again</html>")
					return
				}
				status, savedComment, timestamp := "1", comment, oldTimestamp
				if isReadback {
					if r.Header.Get("Cache-Control") != "no-cache" {
						t.Error("readback did not request cache revalidation")
					}
					timestamp = newTimestamp
					if test.readbackStatus != "" {
						status = test.readbackStatus
					}
					if test.readbackComment != "" {
						savedComment = test.readbackComment
					}
					if test.readbackTime != "" {
						timestamp = test.readbackTime
					}
				}
				action := "/status/new_status"
				if test.fakeSave {
					action = "/unrelated"
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				fmt.Fprintf(w, `<select id="statusSelector"><option value="%s" selected>Test status</option></select>
<textarea id="currentStatusComment">%s</textarea><span id="currentStatusTimestamp">%s</span>
<button id="statusEvenBtn">Save</button><script>
document.getElementById('statusEvenBtn').onclick = () => {
 fetch('%s', {method:'POST',body:new URLSearchParams({status:document.getElementById('statusSelector').value,comment:document.getElementById('currentStatusComment').value})})
 .then(r=>r.json()).then(data=>{document.getElementById('currentStatusTimestamp').innerHTML=data.last_updated || '';}).catch(()=>{});
};</script>`, status, html.EscapeString(savedComment), timestamp, action)
			}))
			defer server.Close()
			page, err := browser.NewPage()
			if err != nil {
				t.Fatal(err)
			}
			defer page.Close()
			page.SetDefaultTimeout(1500)
			page.SetDefaultNavigationTimeout(3000)
			err = saveAndVerifyParade(page, server.URL+"/status")
			if test.wantVerified && err != nil {
				t.Fatalf("persisted save not verified: %v", err)
			}
			if !test.wantVerified && !errors.Is(err, errSaveUnverified) {
				t.Fatalf("expected an unverified result, got %v", err)
			}
			wantPosts := int32(1)
			if test.fakeSave {
				wantPosts = 0
			}
			if posts.Load() != wantPosts {
				t.Fatalf("save was repeated or missing: got %d posts, want %d", posts.Load(), wantPosts)
			}
			if test.wantVerified && reads.Load() < 2 {
				t.Fatal("success reported without a fresh readback")
			}
		})
	}
}

func TestUnverifiedSaveStatus(t *testing.T) {
	recordResult(fmt.Errorf("%w: simulated stale readback", errSaveUnverified))
	_, result := currentStatus()
	if !strings.HasPrefix(result, "Save not verified;") {
		t.Fatalf("tray incorrectly described an uncertain save: %s", result)
	}
}
