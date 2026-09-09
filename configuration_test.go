package main

import (
	"context"
	"strings"
	"testing"
)

func TestUnconfiguredBotStopsBeforeBrowserSetup(t *testing.T) {
	previousUser, previousPassword := username, password
	t.Cleanup(func() { username, password = previousUser, previousPassword })
	for _, credentials := range [][2]string{{"", ""}, {"user", ""}, {"", "password"}, {"YOUR_RMC_ID", "YOUR_RMC_PASSWORD"}} {
		username, password = credentials[0], credentials[1]
		if err := runBot(context.Background(), true, nil); err == nil || !strings.Contains(err.Error(), "setup required") {
			t.Fatalf("expected setup error before any browser or network work, got %v", err)
		}
	}
}

func TestUnconfiguredNotificationsDoNotSend(t *testing.T) {
	previous := username
	t.Cleanup(func() { username = previous })
	username = ""
	notify("Test", "Must not send")
	for _, value := range []string{"", " ", "YOUR_RMC_ID"} {
		username = value
		if err := sendNtfy("Test", "Must not send"); err == nil {
			t.Fatalf("accepted an unconfigured username %q", value)
		}
	}
}

func TestNotificationTopicFollowsUsername(t *testing.T) {
	previous := username
	t.Cleanup(func() { username = previous })
	for _, value := range []string{"ID_EXAMPLE1", "ID_EXAMPLE2"} {
		username = value
		endpoint, err := notificationURL()
		if err != nil || endpoint != "https://ntfy.sh/paradebot-"+value {
			t.Fatalf("topic did not follow username: %q, %v", endpoint, err)
		}
	}
}

func TestInvalidTimeWindowsAreRejected(t *testing.T) {
	oldStartHour, oldStartMinute, oldEndHour, oldEndMinute := startHour, startMinute, endHour, endMinute
	t.Cleanup(func() {
		startHour, startMinute, endHour, endMinute = oldStartHour, oldStartMinute, oldEndHour, oldEndMinute
	})
	for _, window := range [][4]int{{-1, 0, 7, 0}, {5, 60, 7, 0}, {5, 30, 24, 0}, {5, 30, 7, -1}, {7, 0, 7, 0}, {23, 0, 5, 0}} {
		startHour, startMinute, endHour, endMinute = window[0], window[1], window[2], window[3]
		if err := validateTimeWindow(); err == nil {
			t.Fatalf("invalid window accepted: %v", window)
		}
	}
	startHour, startMinute, endHour, endMinute = 5, 30, 7, 0
	if err := validateTimeWindow(); err != nil {
		t.Fatal(err)
	}
}
